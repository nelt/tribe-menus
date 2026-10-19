package main

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
)

// Response times of the login (plan revue-securite, "Ce qui n'a pas pu être vérifié"):
// does the time of an answer tell a member from another address, or an existing tribe
// from another? Requests of each case alternate, at the pace the limits leave to an
// attacker from one connection: POST sessions without the cookie of a code request is not
// limited; POST login-codes is, ten by hour and by IP (ENF-01).

const (
	// sessionsPace separates the requests on POST sessions.
	sessionsPace = 200 * time.Millisecond
	// codesByRound is the number of code requests by hour, under the limit by IP, three by
	// case. The limit by address (three in fifteen minutes) allows three for one member.
	codesByRound = 9
	// roundWait separates the rounds of code requests: past the window of the limit by IP.
	roundWait = 61 * time.Minute
	// minVerdictSamples is the number of samples by case below which no verdict is given.
	minVerdictSamples = 20
	// maxGap is the distance to 0.5 of the probability that a time of one case exceeds a
	// time of the other, above which the gap is to examine.
	maxGap = 0.15
)

// sample is the time and status of a request.
type sample struct {
	took   time.Duration
	status int
}

// timingCase is a kind of request whose times are compared.
type timingCase struct {
	label string
	send  func(ctx context.Context, i int) (sample, error)
}

func (p *prober) timing(ctx context.Context) {
	other := "sonde-" + randomHex(6)
	run := randomHex(4)
	// A first request opens the connection, reused by the others, as an attacker would.
	if _, _, err := p.get(ctx, p.url("/tribes/"+p.tribe+"/")); err != nil {
		p.fail("application: %s", p.errorText(err))
		return
	}

	p.section(fmt.Sprintf("POST sessions, without a code request: %d requests by case, every %s", p.samples, sessionsPace))
	sessions := []timingCase{
		{"member, existing tribe", func(ctx context.Context, i int) (sample, error) {
			return p.openSession(ctx, p.tribe, p.members[i%len(p.members)])
		}},
		{"other address, existing tribe", func(ctx context.Context, i int) (sample, error) {
			return p.openSession(ctx, p.tribe, fmt.Sprintf("sonde-%s-%d@example.org", run, i))
		}},
		{"member address, other tribe", func(ctx context.Context, i int) (sample, error) {
			return p.openSession(ctx, other, p.members[i%len(p.members)])
		}},
	}
	results, ok := p.measure(ctx, sessions, p.samples, sessionsPace)
	if ok {
		p.compare(results, sessions[0].label, sessions[1].label)
		p.compare(results, sessions[0].label, sessions[2].label)
	}

	p.section(fmt.Sprintf("POST login-codes: %d requests an hour, %d hour(s); a member receives an email", codesByRound, p.rounds))
	codes := []timingCase{
		{"member, existing tribe", func(ctx context.Context, i int) (sample, error) {
			return p.timeCode(ctx, p.tribe, p.members[i%len(p.members)])
		}},
		{"other address, existing tribe", func(ctx context.Context, i int) (sample, error) {
			return p.timeCode(ctx, p.tribe, fmt.Sprintf("sonde-%s-c%d@example.org", run, i))
		}},
		{"other address, other tribe", func(ctx context.Context, i int) (sample, error) {
			return p.timeCode(ctx, other, fmt.Sprintf("sonde-%s-o%d@example.org", run, i))
		}},
	}
	all := map[string][]sample{}
	for round := range p.rounds {
		if round > 0 {
			p.info("waiting %s for the next hour of requests", roundWait)
			select {
			case <-ctx.Done():
				return
			case <-time.After(roundWait):
			}
		}
		results, ok := p.measure(ctx, codes, codesByRound/len(codes), sessionsPace)
		if !ok {
			return
		}
		for label, s := range results {
			all[label] = append(all[label], s...)
		}
	}
	p.compare(all, codes[0].label, codes[1].label)
	p.compare(all, codes[1].label, codes[2].label)
}

// measure sends n requests of each case, alternating, and returns the samples by case. The
// statuses must not differ between cases: a different answer tells more than a time.
func (p *prober) measure(ctx context.Context, cases []timingCase, n int, pace time.Duration) (map[string][]sample, bool) {
	results := map[string][]sample{}
	for i := range n {
		for _, c := range cases {
			s, err := c.send(ctx, i)
			if err != nil {
				p.fail("%s: %s", c.label, p.errorText(err))
				return nil, false
			}
			if s.status == http.StatusTooManyRequests {
				p.fail("%s: refused by a limit (429) at request %d: budget of the hour already used", c.label, i+1)
				return nil, false
			}
			results[c.label] = append(results[c.label], s)
			select {
			case <-ctx.Done():
				return nil, false
			case <-time.After(pace):
			}
		}
	}
	statuses := map[string]string{}
	for _, c := range cases {
		statuses[c.label] = statusSet(results[c.label])
		st := summarize(durations(results[c.label]))
		p.info("%s: n %d, median %s, quartiles %s to %s, statuses %s", c.label, st.n,
			ms(st.median), ms(st.q1), ms(st.q3), statuses[c.label])
	}
	same := len(slices.Compact(slices.Sorted(maps.Values(statuses)))) == 1
	p.check(same, "same statuses for every case")
	return results, true
}

// compare writes the verdict on the times of two cases.
func (p *prober) compare(results map[string][]sample, a, b string) {
	x, y := durations(results[a]), durations(results[b])
	gap := superiority(x, y)
	diff := summarize(x).median - summarize(y).median
	line := fmt.Sprintf("%s / %s: medians apart by %s, P(first slower) %.2f", a, b, ms(diff), gap)
	switch {
	case len(x) < minVerdictSamples || len(y) < minVerdictSamples:
		p.info("%s; fewer than %d samples by case, no verdict", line, minVerdictSamples)
	case gap > 0.5-maxGap && gap < 0.5+maxGap:
		p.ok("%s: no measurable gap", line)
	default:
		p.fail("%s: gap to examine (docs/traitement-des-vulnerabilites.md)", line)
	}
}

func (p *prober) openSession(ctx context.Context, slug, email string) (sample, error) {
	body := fmt.Sprintf(`{"email":%q,"code":"00000000","installedApp":false}`, email)
	return p.timed(ctx, p.url("/tribes/"+slug+"/api/sessions"), body)
}

func (p *prober) timeCode(ctx context.Context, slug, email string) (sample, error) {
	return p.timed(ctx, p.url("/tribes/"+slug+"/api/login-codes"), fmt.Sprintf(`{"email":%q}`, email))
}

// timed sends a POST and returns the time until the end of the answer.
func (p *prober) timed(ctx context.Context, rawURL, body string) (sample, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(body))
	if err != nil {
		return sample{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return sample{}, err
	}
	_, err = io.Copy(io.Discard, resp.Body)
	took := time.Since(start)
	_ = resp.Body.Close()
	if err != nil {
		return sample{}, err
	}
	return sample{took: took, status: resp.StatusCode}, nil
}

func durations(samples []sample) []time.Duration {
	d := make([]time.Duration, len(samples))
	for i, s := range samples {
		d[i] = s.took
	}
	return d
}

func statusSet(samples []sample) string {
	var codes []int
	for _, s := range samples {
		codes = append(codes, s.status)
	}
	slices.Sort(codes)
	return joinInts(slices.Compact(codes))
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1f ms", float64(d.Microseconds())/1000)
}
