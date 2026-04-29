package config

import "testing"

func TestCompile(t *testing.T) {
	cfg := Config{
		Routes: []Route{
			{
				Match:  "PATCH /receivables",
				Target: "http://service-a.internal",
			},
			{
				Match:       "PATCH /receivables",
				HeaderName:  "X-Header-Redirect",
				HeaderValue: "journey-x",
				Target:      "http://service-x.internal",
			},
			{
				Match:  "PATCH /receivables/*",
				Target: "http://service-b.internal",
			},
		},
	}

	compiled, err := Compile(cfg)
	if err != nil {
		t.Fatalf("compile returned error: %v", err)
	}
	if len(compiled.Routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(compiled.Routes))
	}
	for _, r := range compiled.Routes {
		if r.Transport == nil {
			t.Fatalf("expected transport on each compiled route")
		}
	}
	if !compiled.Routes[2].IsWildcard {
		t.Fatalf("expected route 2 to be wildcard")
	}
}

func TestResolveRewrite(t *testing.T) {
	route := CompiledRoute{
		Method:              "PATCH",
		PathPattern:         "/receivables/*",
		IsWildcard:          true,
		WildcardBase:        "/receivables",
		RewriteMethod:       "PUT",
		RewritePath:         "/v2/receivables/*",
		RewriteIsWildcard:   true,
		RewriteWildcardBase: "/v2/receivables",
	}

	method, path := route.ResolveRewrite("PATCH", "/receivables/123/items")
	if method != "PUT" {
		t.Fatalf("expected rewrite method PUT, got %s", method)
	}
	if path != "/v2/receivables/123/items" {
		t.Fatalf("expected wildcard rewrite path preserved suffix, got %s", path)
	}
}

func TestCompileMidSegmentWildcard(t *testing.T) {
	cfg := Config{
		Routes: []Route{
			{
				Match:   "DELETE /receivables/*/items",
				Rewrite: "DELETE /v2/receivables/*/product_items",
				Target:  "http://service-a.internal",
			},
		},
	}

	compiled, err := Compile(cfg)
	if err != nil {
		t.Fatalf("compile returned error: %v", err)
	}
	r := compiled.Routes[0]
	if !r.IsWildcard {
		t.Fatalf("expected IsWildcard=true")
	}
	if r.WildcardBase != "/receivables" {
		t.Fatalf("expected WildcardBase=/receivables, got %q", r.WildcardBase)
	}
	if r.WildcardSuffix != "/items" {
		t.Fatalf("expected WildcardSuffix=/items, got %q", r.WildcardSuffix)
	}
	if r.RewriteWildcardBase != "/v2/receivables" {
		t.Fatalf("expected RewriteWildcardBase=/v2/receivables, got %q", r.RewriteWildcardBase)
	}
	if r.RewriteWildcardSuffix != "/product_items" {
		t.Fatalf("expected RewriteWildcardSuffix=/product_items, got %q", r.RewriteWildcardSuffix)
	}
}

func TestResolveRewriteMidSegmentWildcard(t *testing.T) {
	route := CompiledRoute{
		Method:                "DELETE",
		PathPattern:           "/receivables/*/items",
		IsWildcard:            true,
		WildcardBase:          "/receivables",
		WildcardSuffix:        "/items",
		RewriteMethod:         "DELETE",
		RewritePath:           "/v2/receivables/*/product_items",
		RewriteIsWildcard:     true,
		RewriteWildcardBase:   "/v2/receivables",
		RewriteWildcardSuffix: "/product_items",
	}

	method, path := route.ResolveRewrite("DELETE", "/receivables/abc-123/items")
	if method != "DELETE" {
		t.Fatalf("expected method DELETE, got %s", method)
	}
	if path != "/v2/receivables/abc-123/product_items" {
		t.Fatalf("expected /v2/receivables/abc-123/product_items, got %s", path)
	}
}

func TestCompileRejectsInvalidHeaderPair(t *testing.T) {
	cfg := Config{
		Routes: []Route{
			{
				Match:      "PATCH /receivables",
				HeaderName: "X-Header-Redirect",
				Target:     "http://service-a.internal",
			},
		},
	}

	_, err := Compile(cfg)
	if err == nil {
		t.Fatalf("expected error for incomplete header rule")
	}
}
