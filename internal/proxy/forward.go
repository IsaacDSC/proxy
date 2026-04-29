package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/IsaacDSC/proxy/internal/config"
)

// hopByHopHeaders lists HTTP/1.1 hop-by-hop headers (RFC 7230). They describe only the
// connection between client and this proxy, not the separate TCP hop to the upstream.
// Forwarding them would leak wrong connection semantics (e.g. Connection, Upgrade),
// proxy-specific auth (Proxy-*), or framing meant for the inbound body (Transfer-Encoding).
// We strip them from the outbound request before contacting the backend.
var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

// Forward proxies r according to route (rewrite rules, transport, target).
func Forward(route *config.CompiledRoute, w http.ResponseWriter, r *http.Request) error {
	rewriteMethod, rewritePath := route.ResolveRewrite(r.Method, r.URL.Path)

	targetURL, err := url.Parse(strings.TrimSpace(route.Target))
	if err != nil {
		return err
	}

	requestPath := r.URL.Path
	if strings.TrimSpace(rewritePath) != "" {
		requestPath = rewritePath
	}

	targetURL.Path = singleJoiningSlash(targetURL.Path, requestPath)
	targetURL.RawQuery = r.URL.RawQuery

	method := r.Method
	if strings.TrimSpace(rewriteMethod) != "" {
		method = rewriteMethod
	}

	var proxyErr error
	rp := &httputil.ReverseProxy{
		Director: func(outReq *http.Request) {
			outReq.URL.Scheme = targetURL.Scheme
			outReq.URL.Host = targetURL.Host
			outReq.URL.Path = targetURL.Path
			outReq.URL.RawPath = targetURL.RawPath
			outReq.URL.RawQuery = targetURL.RawQuery
			outReq.Host = targetURL.Host
			outReq.Method = method
			removeHopByHopHeaders(outReq.Header)
			applyStaticHeaders(outReq.Header, route.Headers)
			applyRenameHeaders(outReq.Header, route.RenameHeaders)
		},
		Transport: route.Transport,
		ErrorHandler: func(_ http.ResponseWriter, _ *http.Request, err error) {
			proxyErr = err
		},
	}
	rp.ServeHTTP(w, r)
	if proxyErr != nil {
		return proxyErr
	}
	if err := r.Context().Err(); errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

// removeHopByHopHeaders deletes hop-by-hop header fields from headers before forwarding.
func removeHopByHopHeaders(headers http.Header) {
	for key := range headers {
		if _, ok := hopByHopHeaders[http.CanonicalHeaderKey(key)]; ok {
			headers.Del(key)
		}
	}
}

// applyStaticHeaders sets each key-value pair from headers onto the outbound request,
// overwriting any existing value for that header.
func applyStaticHeaders(h http.Header, headers map[string]string) {
	for k, v := range headers {
		h.Set(k, v)
	}
}

// applyRenameHeaders renames headers according to the provided rules.
// For each rule it copies the value from the current header name to the new one and deletes the old key.
// Rules with a missing current header are silently skipped.
func applyRenameHeaders(headers http.Header, rules []config.RenameHeaderRule) {
	for _, rule := range rules {
		current := http.CanonicalHeaderKey(rule.Current)
		newKey := http.CanonicalHeaderKey(rule.Replace)
		if values := headers[current]; len(values) > 0 {
			headers[newKey] = values
			delete(headers, current)
		}
	}
}

func singleJoiningSlash(base string, target string) string {
	switch {
	case strings.HasSuffix(base, "/") && strings.HasPrefix(target, "/"):
		return base + target[1:]
	case !strings.HasSuffix(base, "/") && !strings.HasPrefix(target, "/"):
		return base + "/" + target
	default:
		return base + target
	}
}
