package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	if err := Forward(backend.Client(), backend.URL, rec, req); err != nil {
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

func TestForwardWithRewrite(t *testing.T) {
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
	if err := ForwardWithRewrite(backend.Client(), backend.URL, http.MethodPut, "/v2/receivables/123", rec, req); err != nil {
		t.Fatalf("forward with rewrite error: %v", err)
	}

	resp := rec.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}
}
