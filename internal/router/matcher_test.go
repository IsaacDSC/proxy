package router

import (
	"net/http"
	"testing"

	"github.com/IsaacDSC/proxy/internal/config"
)

func TestMatchRoute(t *testing.T) {
	routes := []config.CompiledRoute{
		{
			Route: config.Route{
				Match:  "PATCH /receivables",
				Target: "http://service-a.internal",
			},
			Method:      "PATCH",
			PathPattern: "/receivables",
			Index:       0,
		},
		{
			Route: config.Route{
				Match:  "PATCH /receivables/*",
				Target: "http://service-b.internal",
			},
			Method:       "PATCH",
			PathPattern:  "/receivables/*",
			IsWildcard:   true,
			WildcardBase: "/receivables",
			Index:        1,
		},
		{
			Route: config.Route{
				Match:       "PATCH /receivables",
				HeaderName:  "X-Header-Redirect",
				HeaderValue: "journey-x",
				Target:      "http://service-x.internal",
			},
			Method:      "PATCH",
			PathPattern: "/receivables",
			Index:       2,
		},
	}
	matcher := NewMatcher(routes)

	tests := []struct {
		name       string
		method     string
		path       string
		headers    http.Header
		wantTarget string
	}{
		{
			name:       "match exact without header",
			method:     "PATCH",
			path:       "/receivables",
			headers:    http.Header{},
			wantTarget: "http://service-a.internal",
		},
		{
			name:       "match wildcard",
			method:     "PATCH",
			path:       "/receivables/123",
			headers:    http.Header{},
			wantTarget: "http://service-b.internal",
		},
		{
			name:   "header route has priority",
			method: "PATCH",
			path:   "/receivables",
			headers: http.Header{
				"X-Header-Redirect": []string{"journey-x"},
			},
			wantTarget: "http://service-x.internal",
		},
		{
			name:   "header mismatch falls back",
			method: "PATCH",
			path:   "/receivables",
			headers: http.Header{
				"X-Header-Redirect": []string{"unknown"},
			},
			wantTarget: "http://service-a.internal",
		},
		{
			name:       "no route returns nil",
			method:     "GET",
			path:       "/does-not-exist",
			headers:    http.Header{},
			wantTarget: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			match := matcher.MatchRoute(tc.method, tc.path, tc.headers)
			if tc.wantTarget == "" {
				if match != nil {
					t.Fatalf("expected nil match, got %+v", *match)
				}
				return
			}

			if match == nil {
				t.Fatalf("expected route %q, got nil", tc.wantTarget)
			}
			if match.Target != tc.wantTarget {
				t.Fatalf("expected target %q, got %q", tc.wantTarget, match.Target)
			}
		})
	}
}
