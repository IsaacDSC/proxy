package router

import (
	"net/http"
	"strings"

	"github.com/IsaacDSC/proxy/internal/config"
)

type Matcher struct {
	exactByMethodPath map[string]map[string][]*config.CompiledRoute
	wildcardsByMethod map[string][]*config.CompiledRoute
}

func NewMatcher(routes []config.CompiledRoute) *Matcher {
	matcher := &Matcher{
		exactByMethodPath: make(map[string]map[string][]*config.CompiledRoute),
		wildcardsByMethod: make(map[string][]*config.CompiledRoute),
	}

	for idx := range routes {
		route := &routes[idx]
		method := strings.ToUpper(route.Method)

		if route.IsWildcard {
			matcher.wildcardsByMethod[method] = append(matcher.wildcardsByMethod[method], route)
			continue
		}

		if _, ok := matcher.exactByMethodPath[method]; !ok {
			matcher.exactByMethodPath[method] = make(map[string][]*config.CompiledRoute)
		}
		matcher.exactByMethodPath[method][route.PathPattern] = append(matcher.exactByMethodPath[method][route.PathPattern], route)
	}

	return matcher
}

func (m *Matcher) MatchRoute(method string, path string, headers http.Header) *config.CompiledRoute {
	var headerExact *config.CompiledRoute
	var headerWildcard *config.CompiledRoute
	var plainExact *config.CompiledRoute
	var plainWildcard *config.CompiledRoute

	normalizedMethod := strings.ToUpper(method)

	if methodPaths, ok := m.exactByMethodPath[normalizedMethod]; ok {
		for _, route := range methodPaths[path] {
			hasHeaderRule := route.HeaderName != "" && route.HeaderValue != ""
			if hasHeaderRule {
				if headers.Get(route.HeaderName) != route.HeaderValue {
					continue
				}
				if headerExact == nil {
					headerExact = route
				}
				continue
			}

			if plainExact == nil {
				plainExact = route
			}
		}
	}

	for _, route := range m.wildcardsByMethod[normalizedMethod] {
		if !matchPath(*route, path) {
			continue
		}

		hasHeaderRule := route.HeaderName != "" && route.HeaderValue != ""
		if hasHeaderRule {
			if headers.Get(route.HeaderName) != route.HeaderValue {
				continue
			}
			if headerWildcard == nil {
				headerWildcard = route
			}
			continue
		}

		if plainWildcard == nil {
			plainWildcard = route
		}
	}

	switch {
	case headerExact != nil:
		return headerExact
	case headerWildcard != nil:
		return headerWildcard
	case plainExact != nil:
		return plainExact
	case plainWildcard != nil:
		return plainWildcard
	default:
		return nil
	}
}

func MatchRoute(routes []config.CompiledRoute, method string, path string, headers http.Header) *config.CompiledRoute {
	return NewMatcher(routes).MatchRoute(method, path, headers)
}

func matchPath(route config.CompiledRoute, requestPath string) bool {
	if !route.IsWildcard {
		return route.PathPattern == requestPath
	}

	base := route.WildcardBase
	if requestPath == base {
		return false
	}
	return strings.HasPrefix(requestPath, base+"/")
}
