package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Transport TransportConfig `json:"transport,omitempty"`
	Routes    []Route         `json:"routes"`
}

type RenameHeaderRule struct {
	Current string `json:"current"`
	New     string `json:"new"`
}

type Route struct {
	Match         string              `json:"match"`
	Target        string              `json:"target"`
	HeaderName    string              `json:"header_name,omitempty"`
	HeaderValue   string              `json:"header_value,omitempty"`
	Rewrite       string              `json:"rewrite,omitempty"`
	Headers       map[string]string   `json:"headers,omitempty"`
	RenameHeaders []RenameHeaderRule  `json:"rename_header,omitempty"`
	Transport     TransportConfig     `json:"transport,omitempty"`
}

type CompiledRoute struct {
	Route
	Transport            http.RoundTripper
	Method               string
	PathPattern          string
	IsWildcard           bool
	WildcardBase         string
	WildcardSuffix       string
	RewriteMethod        string
	RewritePath          string
	RewriteIsWildcard    bool
	RewriteWildcardBase  string
	RewriteWildcardSuffix string
	Index                int
}

type CompiledConfig struct {
	Routes []CompiledRoute
}

func Load(path string) (CompiledConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CompiledConfig{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return CompiledConfig{}, fmt.Errorf("parse config json: %w", err)
	}

	return Compile(cfg)
}

func Compile(cfg Config) (CompiledConfig, error) {
	if len(cfg.Routes) == 0 {
		return CompiledConfig{}, fmt.Errorf("config must contain at least one route")
	}

	compiled := make([]CompiledRoute, 0, len(cfg.Routes))
	for i, route := range cfg.Routes {
		method, path, err := parseMatch(route.Match)
		if err != nil {
			return CompiledConfig{}, fmt.Errorf("route %d invalid match: %w", i, err)
		}
		if strings.TrimSpace(route.Target) == "" {
			return CompiledConfig{}, fmt.Errorf("route %d target is required", i)
		}
		if (route.HeaderName == "") != (route.HeaderValue == "") {
			return CompiledConfig{}, fmt.Errorf("route %d header_name and header_value must be provided together", i)
		}
		rewriteMethod, rewritePath, err := parseRewrite(route.Rewrite)
		if err != nil {
			return CompiledConfig{}, fmt.Errorf("route %d invalid rewrite: %w", i, err)
		}

		isWildcard := strings.Contains(path, "/*")
		wildcardBase, wildcardSuffix := parseWildcardParts(path)
		rewriteIsWildcard := strings.Contains(rewritePath, "/*")
		rewriteWildcardBase, rewriteWildcardSuffix := parseWildcardParts(rewritePath)

		merged := mergeTransport(cfg.Transport, route.Transport)
		rt, err := NewHTTPTransport(merged)
		if err != nil {
			return CompiledConfig{}, fmt.Errorf("route %d transport: %w", i, err)
		}

		compiled = append(compiled, CompiledRoute{
			Route:                 route,
			Transport:             rt,
			Method:                method,
			PathPattern:           path,
			IsWildcard:            isWildcard,
			WildcardBase:          wildcardBase,
			WildcardSuffix:        wildcardSuffix,
			RewriteMethod:         rewriteMethod,
			RewritePath:           rewritePath,
			RewriteIsWildcard:     rewriteIsWildcard,
			RewriteWildcardBase:   rewriteWildcardBase,
			RewriteWildcardSuffix: rewriteWildcardSuffix,
			Index:                 i,
		})
	}

	return CompiledConfig{Routes: compiled}, nil
}

func parseRewrite(rewrite string) (method string, path string, err error) {
	trimmed := strings.TrimSpace(rewrite)
	if trimmed == "" {
		return "", "", nil
	}

	method, path, err = parseMatch(trimmed)
	if err != nil {
		return "", "", err
	}

	return method, path, nil
}

func (r CompiledRoute) ResolveRewrite(requestMethod string, requestPath string) (method string, path string) {
	method = requestMethod
	path = requestPath

	if r.RewriteMethod != "" {
		method = r.RewriteMethod
	}
	if r.RewritePath == "" {
		return method, path
	}

	if r.IsWildcard && r.RewriteIsWildcard {
		if strings.HasPrefix(requestPath, r.WildcardBase+"/") {
			rest := strings.TrimPrefix(requestPath, r.WildcardBase)
			if r.WildcardSuffix != "" {
				// mid-segment: extract just the wildcard segment by stripping the suffix
				wildcardPart := strings.TrimSuffix(rest, r.WildcardSuffix)
				return method, r.RewriteWildcardBase + wildcardPart + r.RewriteWildcardSuffix
			}
			return method, r.RewriteWildcardBase + rest
		}
	}

	return method, r.RewritePath
}

// parseWildcardParts splits a path containing "/*" into its base and suffix components.
// For a trailing wildcard "/prefix/*" it returns ("/prefix", "").
// For a mid-segment wildcard "/prefix/*/suffix" it returns ("/prefix", "/suffix").
// For paths without a wildcard it returns ("", "").
func parseWildcardParts(path string) (base string, suffix string) {
	idx := strings.Index(path, "/*/")
	if idx >= 0 {
		return path[:idx], path[idx+2:]
	}
	if strings.HasSuffix(path, "/*") {
		return strings.TrimSuffix(path, "/*"), ""
	}
	return "", ""
}

func parseMatch(match string) (method string, path string, err error) {
	trimmed := strings.TrimSpace(match)
	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("match must be '<METHOD> <PATH>'")
	}

	method = strings.ToUpper(strings.TrimSpace(parts[0]))
	path = strings.TrimSpace(parts[1])

	if method == "" || path == "" {
		return "", "", fmt.Errorf("method and path are required")
	}
	if !strings.HasPrefix(path, "/") {
		return "", "", fmt.Errorf("path must start with '/'")
	}
	if strings.Count(path, "*") > 1 {
		return "", "", fmt.Errorf("at most one wildcard '*' is supported")
	}
	if strings.Contains(path, "*") {
		idx := strings.Index(path, "*")
		if idx == 0 || path[idx-1] != '/' {
			return "", "", fmt.Errorf("wildcard '*' must be a full path segment (e.g. /prefix/* or /prefix/*/suffix)")
		}
		if idx < len(path)-1 && path[idx+1] != '/' {
			return "", "", fmt.Errorf("wildcard '*' must be a full path segment (e.g. /prefix/* or /prefix/*/suffix)")
		}
	}

	return method, path, nil
}
