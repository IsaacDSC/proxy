package router

import (
	"net/http"
	"net/http/httptest"
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
			name:   "header mismatch returns nil",
			method: "PATCH",
			path:   "/receivables",
			headers: http.Header{
				"X-Header-Redirect": []string{"unknown"},
			},
			wantTarget: "",
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
			req := httptest.NewRequest(tc.method, "http://example.com"+tc.path, nil)
			req.Header = tc.headers.Clone()
			match := matcher.MatchRoute(req)
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

func TestMatchRouteMidSegmentWildcard(t *testing.T) {
	routes := []config.CompiledRoute{
		{
			Route: config.Route{
				Match:  "DELETE /receivables/*/items",
				Target: "http://service-items.internal",
			},
			Method:         "DELETE",
			PathPattern:    "/receivables/*/items",
			IsWildcard:     true,
			WildcardBase:   "/receivables",
			WildcardSuffix: "/items",
			Index:          0,
		},
		{
			Route: config.Route{
				Match:  "DELETE /receivables/*",
				Target: "http://service-delete.internal",
			},
			Method:       "DELETE",
			PathPattern:  "/receivables/*",
			IsWildcard:   true,
			WildcardBase: "/receivables",
			Index:        1,
		},
	}
	matcher := NewMatcher(routes)

	tests := []struct {
		name       string
		method     string
		path       string
		wantTarget string
	}{
		{
			name:       "mid-segment wildcard matches /receivables/{id}/items",
			method:     "DELETE",
			path:       "/receivables/abc-123/items",
			wantTarget: "http://service-items.internal",
		},
		{
			name:       "trailing wildcard still matches /receivables/{id}",
			method:     "DELETE",
			path:       "/receivables/abc-123",
			wantTarget: "http://service-delete.internal",
		},
		{
			name:       "mid-segment takes priority over trailing wildcard",
			method:     "DELETE",
			path:       "/receivables/xyz/items",
			wantTarget: "http://service-items.internal",
		},
		{
			name:       "no match when suffix differs",
			method:     "DELETE",
			path:       "/receivables/xyz/other",
			wantTarget: "http://service-delete.internal",
		},
		{
			// /receivables/items has no second segment so it cannot match the
			// mid-segment pattern, but it does match the trailing wildcard.
			name:       "falls through to trailing wildcard when suffix looks like segment",
			method:     "DELETE",
			path:       "/receivables/items",
			wantTarget: "http://service-delete.internal",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://example.com"+tc.path, nil)
			match := matcher.MatchRoute(req)
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
