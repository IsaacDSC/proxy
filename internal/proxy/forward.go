package proxy

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

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

func Forward(client *http.Client, targetBase string, w http.ResponseWriter, r *http.Request) error {
	return ForwardWithRewrite(client, targetBase, "", "", w, r)
}

func ForwardWithRewrite(client *http.Client, targetBase string, rewriteMethod string, rewritePath string, w http.ResponseWriter, r *http.Request) error {
	targetURL, err := url.Parse(strings.TrimSpace(targetBase))
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

	req, err := http.NewRequestWithContext(r.Context(), method, targetURL.String(), r.Body)
	if err != nil {
		return err
	}

	copyHeaders(req.Header, r.Header)
	removeHopByHopHeaders(req.Header)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	removeHopByHopHeaders(w.Header())

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	return err
}

func copyHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, v := range values {
			dst.Add(key, v)
		}
	}
}

func removeHopByHopHeaders(headers http.Header) {
	for key := range headers {
		if _, ok := hopByHopHeaders[http.CanonicalHeaderKey(key)]; ok {
			headers.Del(key)
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
