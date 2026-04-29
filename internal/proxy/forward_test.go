package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/IsaacDSC/proxy/internal/config"
)

func TestForward(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected method PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/receivables/123" {
			t.Fatalf("expected path /receivables/123, got %s", r.URL.Path)
		}
		if r.URL.RawQuery != "foo=bar" {
			t.Fatalf("expected query foo=bar, got %s", r.URL.RawQuery)
		}
		if r.Header.Get("X-Header-Redirect") != "journey-x" {
			t.Fatalf("expected header journey-x")
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != `{"id":"123"}` {
			t.Fatalf("unexpected body %s", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodPatch, "http://proxy.local/receivables/123?foo=bar", strings.NewReader(`{"id":"123"}`))
	req.Header.Set("X-Header-Redirect", "journey-x")

	rec := httptest.NewRecorder()
	rt := &config.CompiledRoute{
		Route:     config.Route{Target: backend.URL},
		Transport: http.DefaultTransport,
	}
	if err := Forward(rt, rec, req); err != nil {
		t.Fatalf("forward error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected application/json response")
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("unexpected response body %s", string(body))
	}
}

func TestForwardRewrite(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected method PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v2/receivables/123" {
			t.Fatalf("expected path /v2/receivables/123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodPatch, "http://proxy.local/receivables/123", strings.NewReader(`{"id":"123"}`))
	rec := httptest.NewRecorder()
	route := &config.CompiledRoute{
		Route: config.Route{
			Target: backend.URL,
		},
		Transport:     http.DefaultTransport,
		RewriteMethod: http.MethodPut,
		RewritePath:   "/v2/receivables/123",
	}
	if err := Forward(route, rec, req); err != nil {
		t.Fatalf("forward with rewrite error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestForwardWildcardRewrite(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v2/receivables/abc-123" {
			t.Fatalf("expected path /v2/receivables/abc-123, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	// Simulates: DELETE /receivables/* → DELETE /v2/receivables/* preserving suffix.
	req := httptest.NewRequest(http.MethodDelete, "http://proxy.local/receivables/abc-123", nil)
	rec := httptest.NewRecorder()
	route := &config.CompiledRoute{
		Route:               config.Route{Target: backend.URL},
		Transport:           http.DefaultTransport,
		Method:              http.MethodDelete,
		PathPattern:         "/receivables/*",
		IsWildcard:          true,
		WildcardBase:        "/receivables",
		RewriteMethod:       http.MethodDelete,
		RewritePath:         "/v2/receivables/*",
		RewriteIsWildcard:   true,
		RewriteWildcardBase: "/v2/receivables",
	}
	if err := Forward(route, rec, req); err != nil {
		t.Fatalf("forward wildcard rewrite error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestForwardRenameHeader(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Header-Redirect") != "" {
			t.Fatalf("old header X-Header-Redirect should have been removed")
		}
		if r.Header.Get("X-Header-Redirect-New") != "journey-x" {
			t.Fatalf("expected X-Header-Redirect-New to be journey-x, got %q", r.Header.Get("X-Header-Redirect-New"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodPost, "http://proxy.local/v3/receivables", nil)
	req.Header.Set("X-Header-Redirect", "journey-x")

	rec := httptest.NewRecorder()
	route := &config.CompiledRoute{
		Route: config.Route{
			Target: backend.URL,
			RenameHeaders: []config.RenameHeaderRule{
				{Current: "X-Header-Redirect", New: "X-Header-Redirect-New"},
			},
		},
		Transport: http.DefaultTransport,
	}
	if err := Forward(route, rec, req); err != nil {
		t.Fatalf("forward rename header error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestForwardStaticHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Journey") != "journey-x" {
			t.Fatalf("expected X-Journey to be journey-x, got %q", r.Header.Get("X-Journey"))
		}
		if r.Header.Get("X-Other") != "other-value" {
			t.Fatalf("expected X-Other to be other-value, got %q", r.Header.Get("X-Other"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	req := httptest.NewRequest(http.MethodGet, "http://proxy.local/ping", nil)
	rec := httptest.NewRecorder()
	route := &config.CompiledRoute{
		Route: config.Route{
			Target: backend.URL,
			Headers: map[string]string{
				"X-Journey": "journey-x",
				"X-Other":   "other-value",
			},
		},
		Transport: http.DefaultTransport,
	}
	if err := Forward(route, rec, req); err != nil {
		t.Fatalf("forward static headers error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestForwardMidSegmentWildcardRewrite(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/v2/receivables/abc-123/product_items" {
			t.Fatalf("expected path /v2/receivables/abc-123/product_items, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	// Simulates: DELETE /receivables/*/items → DELETE /v2/receivables/*/product_items
	req := httptest.NewRequest(http.MethodDelete, "http://proxy.local/receivables/abc-123/items", nil)
	rec := httptest.NewRecorder()
	route := &config.CompiledRoute{
		Route:                 config.Route{Target: backend.URL},
		Transport:             http.DefaultTransport,
		Method:                http.MethodDelete,
		PathPattern:           "/receivables/*/items",
		IsWildcard:            true,
		WildcardBase:          "/receivables",
		WildcardSuffix:        "/items",
		RewriteMethod:         http.MethodDelete,
		RewritePath:           "/v2/receivables/*/product_items",
		RewriteIsWildcard:     true,
		RewriteWildcardBase:   "/v2/receivables",
		RewriteWildcardSuffix: "/product_items",
	}
	if err := Forward(route, rec, req); err != nil {
		t.Fatalf("forward mid-segment wildcard rewrite error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}
}
