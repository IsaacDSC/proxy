package router

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/IsaacDSC/proxy/internal/config"
)

type Matcher struct {
	expectedHeaders []string
	rules           map[string]config.CompiledRoute
	wildcardRoutes  []config.CompiledRoute
}

func key(route config.CompiledRoute) string {
	return fmt.Sprintf("%s:%s:%s:%s", route.Method, route.PathPattern, route.HeaderName, route.HeaderValue)
}

func NewMatcher(routes []config.CompiledRoute) *Matcher {
	m := &Matcher{
		rules: make(map[string]config.CompiledRoute),
	}

	for _, route := range routes {
		if route.HeaderName != "" && route.HeaderValue != "" {
			m.expectedHeaders = append(m.expectedHeaders, route.HeaderName)
		}

		if route.IsWildcard {
			m.wildcardRoutes = append(m.wildcardRoutes, route)
		}

		m.rules[key(route)] = route
	}

	// Sort wildcard routes so mid-segment patterns (more specific) are checked before
	// trailing wildcards. Within the same specificity, preserve config order by Index.
	sort.Slice(m.wildcardRoutes, func(i, j int) bool {
		iMid := m.wildcardRoutes[i].WildcardSuffix != ""
		jMid := m.wildcardRoutes[j].WildcardSuffix != ""
		if iMid != jMid {
			return iMid
		}
		return m.wildcardRoutes[i].Index < m.wildcardRoutes[j].Index
	})

	return m
}

func (m *Matcher) MatchRoute(r *http.Request) *config.CompiledRoute {
	receivedRoute := config.CompiledRoute{
		Method:      r.Method,
		PathPattern: r.URL.Path,
	}

	//  set headers if present
	if r.Header != nil {
		for _, headerName := range m.expectedHeaders {
			if headerValue := r.Header.Get(headerName); headerValue != "" {
				receivedRoute.HeaderName = headerName
				receivedRoute.HeaderValue = headerValue
			}
		}
	}

	//  exact match (with header, if any recognised header was found in the request)
	if route, ok := m.rules[key(receivedRoute)]; ok {
		return &route
	}

	// fallback: header present but no header-specific route matched — try the
	// same method+path without header requirements.
	if receivedRoute.HeaderName != "" {
		noHeaderKey := fmt.Sprintf("%s:%s::", receivedRoute.Method, receivedRoute.PathPattern)
		if route, ok := m.rules[noHeaderKey]; ok {
			return &route
		}
	}

	// no exact match: check wildcard routes in specificity order
	for _, route := range m.wildcardRoutes {
		if route.Method != r.Method {
			continue
		}
		if !strings.HasPrefix(r.URL.Path, route.WildcardBase+"/") {
			continue
		}
		if route.WildcardSuffix != "" {
			// mid-segment wildcard: path must also end with the suffix and have at
			// least one character as the wildcard segment between base and suffix.
			if !strings.HasSuffix(r.URL.Path, route.WildcardSuffix) {
				continue
			}
			minLen := len(route.WildcardBase) + len(route.WildcardSuffix) + 1
			if len(r.URL.Path) <= minLen {
				continue
			}
		}
		r := route
		return &r
	}

	return nil
}
