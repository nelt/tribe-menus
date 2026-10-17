package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// googleDNS is the JSON API of a public resolver that validates DNSSEC: its AD flag says
// that the answer is signed and checked.
const googleDNS = "https://dns.google/resolve"

// Types of DNS records.
const (
	typeA    = 1
	typeTXT  = 16
	typeAAAA = 28
	typeCAA  = 257
)

type dnsAnswer struct {
	Status int  `json:"Status"`
	AD     bool `json:"AD"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// data returns the data of the answers of a type, TXT strings joined.
func (a dnsAnswer) data(recordType int) []string {
	var out []string
	for _, r := range a.Answer {
		if r.Type != recordType {
			continue
		}
		d := r.Data
		if recordType == typeTXT {
			d = joinTXT(d)
		}
		out = append(out, d)
	}
	return out
}

// joinTXT joins the strings of a TXT record, which the resolver may give quoted.
func joinTXT(d string) string {
	if !strings.HasPrefix(d, `"`) {
		return d
	}
	var b strings.Builder
	for part := range strings.SplitSeq(d, `" "`) {
		b.WriteString(strings.Trim(part, `"`))
	}
	return b.String()
}

func (p *prober) resolve(ctx context.Context, name string, recordType int) (dnsAnswer, error) {
	u := p.dns + "?" + url.Values{"name": {name}, "type": {fmt.Sprint(recordType)}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return dnsAnswer{}, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return dnsAnswer{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return dnsAnswer{}, fmt.Errorf("resolver: status %d", resp.StatusCode)
	}
	var a dnsAnswer
	if err := json.NewDecoder(resp.Body).Decode(&a); err != nil {
		return dnsAnswer{}, fmt.Errorf("resolver: %w", err)
	}
	return a, nil
}

func (p *prober) checkDNS(ctx context.Context) {
	host := p.base.Hostname()
	for _, t := range []struct {
		recordType int
		name       string
	}{{typeA, "A"}, {typeAAAA, "AAAA"}} {
		a, err := p.resolve(ctx, host, t.recordType)
		if err != nil {
			p.fail("%s: %v", t.name, err)
			continue
		}
		p.check(len(a.data(t.recordType)) > 0, "%s record of the host", t.name)
		p.check(a.AD, "%s answer signed and validated (DNSSEC)", t.name)
	}
	p.checkCAA(ctx, host)
	p.checkMail(ctx)
}

// checkCAA looks for the CAA records that apply to the host, from the host up to the mail
// domain (RFC 8659): they must allow Let's Encrypt only (ADR 0007, point 2).
func (p *prober) checkCAA(ctx context.Context, host string) {
	for name := host; ; {
		a, err := p.resolve(ctx, name, typeCAA)
		if err != nil {
			p.fail("CAA: %v", err)
			return
		}
		if records := a.data(typeCAA); len(records) > 0 {
			p.check(caaLetsEncryptOnly(records), "CAA: certificates by letsencrypt.org only (%d record(s))", len(records))
			return
		}
		if name == p.mailDomain || !strings.Contains(name, ".") {
			p.fail("CAA: no record")
			return
		}
		_, name, _ = strings.Cut(name, ".")
	}
}

// caaLetsEncryptOnly succeeds when every issue and issuewild record names letsencrypt.org,
// and there is one at least.
func caaLetsEncryptOnly(records []string) bool {
	issue := 0
	for _, r := range records {
		fields := strings.Fields(r)
		if len(fields) < 3 {
			return false
		}
		tag, value := fields[1], strings.Trim(strings.Join(fields[2:], " "), `"`)
		if tag != "issue" && tag != "issuewild" {
			continue
		}
		domain, _, _ := strings.Cut(value, ";")
		if strings.TrimSpace(domain) != "letsencrypt.org" {
			return false
		}
		issue++
	}
	return issue > 0
}

// checkMail checks SPF, DKIM and DMARC of the mail domain (ADR 0014). It writes their
// presence and policy, not their content.
func (p *prober) checkMail(ctx context.Context) {
	a, err := p.resolve(ctx, p.mailDomain, typeTXT)
	if err != nil {
		p.fail("SPF: %v", err)
	} else {
		var spf []string
		for _, r := range a.data(typeTXT) {
			if strings.HasPrefix(r, "v=spf1 ") {
				spf = append(spf, r)
			}
		}
		good := len(spf) == 1 && (strings.HasSuffix(spf[0], " ~all") || strings.HasSuffix(spf[0], " -all"))
		p.check(good, "SPF: one record, ending in ~all or -all (%d record(s))", len(spf))
		p.check(a.AD, "SPF answer signed and validated (DNSSEC)")
	}

	if p.dkimSelector == "" {
		p.info("DKIM: not checked, -dkim-selector to give (s= in the DKIM-Signature of a received email)")
	} else {
		name := p.dkimSelector + "._domainkey." + p.mailDomain
		a, err := p.resolve(ctx, name, typeTXT)
		if err != nil {
			p.fail("DKIM: %v", err)
		} else {
			found := false
			for _, r := range a.data(typeTXT) {
				if strings.Contains(r, "p=") && !strings.Contains(r, "p=;") {
					found = true
				}
			}
			p.check(found, "DKIM: public key published for the selector")
		}
	}

	a, err = p.resolve(ctx, "_dmarc."+p.mailDomain, typeTXT)
	if err != nil {
		p.fail("DMARC: %v", err)
		return
	}
	var dmarc []string
	for _, r := range a.data(typeTXT) {
		if strings.HasPrefix(r, "v=DMARC1") {
			dmarc = append(dmarc, r)
		}
	}
	if len(dmarc) != 1 {
		p.fail("DMARC: %d record(s), one expected", len(dmarc))
		return
	}
	p.ok("DMARC: one record, policy %s", dmarcPolicy(dmarc[0]))
	p.check(strings.Contains(dmarc[0], "rua=mailto:"), "DMARC: aggregate reports requested (rua)")
}

func dmarcPolicy(record string) string {
	for tag := range strings.SplitSeq(record, ";") {
		name, value, _ := strings.Cut(strings.TrimSpace(tag), "=")
		if name == "p" {
			return value
		}
	}
	return "missing"
}
