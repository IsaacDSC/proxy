package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Routes []Route `json:"routes"`
}

type Route struct {
	Match       string `json:"match"`
	Target      string `json:"target"`
	HeaderName  string `json:"header_name,omitempty"`
	HeaderValue string `json:"header_value,omitempty"`
	Rewrite     string `json:"rewrite,omitempty"`
}

type CompiledRoute struct {
	Route
	Method              string
	PathPattern         string
	IsWildcard          bool
	WildcardBase        string
	RewriteMethod       string
	RewritePath         string
	RewriteIsWildcard   bool
	RewriteWildcardBase string
	Index               int
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

		isWildcard := strings.HasSuffix(path, "/*")
		wildcardBase := ""
		if isWildcard {
			wildcardBase = strings.TrimSuffix(path, "/*")
		}
		rewriteIsWildcard := strings.HasSuffix(rewritePath, "/*")
		rewriteWildcardBase := ""
		if rewriteIsWildcard {
			rewriteWildcardBase = strings.TrimSuffix(rewritePath, "/*")
		}

		compiled = append(compiled, CompiledRoute{
			Route:               route,
			Method:              method,
			PathPattern:         path,
			IsWildcard:          isWildcard,
			WildcardBase:        wildcardBase,
			RewriteMethod:       rewriteMethod,
			RewritePath:         rewritePath,
			RewriteIsWildcard:   rewriteIsWildcard,
			RewriteWildcardBase: rewriteWildcardBase,
			Index:               i,
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
			suffix := strings.TrimPrefix(requestPath, r.WildcardBase)
			return method, r.RewriteWildcardBase + suffix
		}
	}

	return method, r.RewritePath
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
	if strings.Contains(path, "*") && !strings.HasSuffix(path, "/*") {
		return "", "", fmt.Errorf("only trailing wildcard '/*' is supported")
	}

	return method, path, nil
}
