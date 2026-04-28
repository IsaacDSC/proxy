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
