package server

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name        string
		behindProxy bool
		remoteAddr  string
		forwarded   []string
		want        string
		wantWarning bool
	}{
		{name: "TCP", remoteAddr: "192.0.2.1:49152", want: "192.0.2.1"},
		{name: "TCP, IPv6", remoteAddr: "[2001:db8::1]:49152", want: "2001:db8::1"},
		{name: "TCP, forged header ignored", remoteAddr: "192.0.2.1:49152", forwarded: []string{"198.51.100.9"}, want: "192.0.2.1"},
		{name: "proxy, one address", behindProxy: true, remoteAddr: "@", forwarded: []string{"198.51.100.9"}, want: "198.51.100.9"},
		{name: "proxy, the last address wins", behindProxy: true, remoteAddr: "@", forwarded: []string{"203.0.113.1, 198.51.100.9"}, want: "198.51.100.9"},
		{name: "proxy, the last line wins", behindProxy: true, remoteAddr: "@", forwarded: []string{"203.0.113.1", "198.51.100.9"}, want: "198.51.100.9"},
		{name: "proxy, IPv6", behindProxy: true, remoteAddr: "@", forwarded: []string{"203.0.113.1,2001:DB8::A "}, want: "2001:db8::a"},
		{name: "proxy, header missing", behindProxy: true, remoteAddr: "@", want: unknownClientIP, wantWarning: true},
		{name: "proxy, unreadable", behindProxy: true, remoteAddr: "@", forwarded: []string{"198.51.100.9, secret-garbage"}, want: unknownClientIP, wantWarning: true},
		{name: "proxy, empty last", behindProxy: true, remoteAddr: "@", forwarded: []string{"198.51.100.9,"}, want: unknownClientIP, wantWarning: true},
		{name: "proxy, with port", behindProxy: true, remoteAddr: "@", forwarded: []string{"198.51.100.9:1234"}, want: unknownClientIP, wantWarning: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			s := &server{behindProxy: tc.behindProxy, logger: slog.New(slog.NewTextHandler(&logs, nil))}
			req := httptest.NewRequest("POST", "https://localhost/tribes/martin/api/login-codes", nil)
			req.RemoteAddr = tc.remoteAddr
			for _, v := range tc.forwarded {
				req.Header.Add("X-Forwarded-For", v)
			}
			if got := s.clientIP(req); got != tc.want {
				t.Errorf("clientIP = %q, want %q", got, tc.want)
			}
			if warned := strings.Contains(logs.String(), "level=WARN"); warned != tc.wantWarning {
				t.Errorf("warning logged: %v, want %v (%s)", warned, tc.wantWarning, logs.String())
			}
			if strings.Contains(logs.String(), "garbage") {
				t.Errorf("logs quote the header: %s", logs.String())
			}
		})
	}
}
