package config

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// TransportConfig holds settings for http.Transport. Empty/zero fields inherit from
// defaults and parent layers (global transport → route transport).
type TransportConfig struct {
	DialTimeout           string `json:"dial_timeout,omitempty"`
	KeepAlive             string `json:"keep_alive,omitempty"`
	MaxIdleConns          int    `json:"max_idle_conns,omitempty"`
	MaxConnsPerHost       int    `json:"max_conns_per_host,omitempty"`
	IdleConnTimeout       string `json:"idle_conn_timeout,omitempty"`
	ExpectContinueTimeout string `json:"expect_continue_timeout,omitempty"`
}

func defaultTransportConfig() TransportConfig {
	return TransportConfig{
		DialTimeout:           "30s",
		KeepAlive:             "30s",
		MaxIdleConns:          100,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       "90s",
		ExpectContinueTimeout: "1s",
	}
}

func mergeTwo(base, over TransportConfig) TransportConfig {
	out := base
	if over.DialTimeout != "" {
		out.DialTimeout = over.DialTimeout
	}
	if over.KeepAlive != "" {
		out.KeepAlive = over.KeepAlive
	}
	if over.MaxIdleConns != 0 {
		out.MaxIdleConns = over.MaxIdleConns
	}
	if over.MaxConnsPerHost != 0 {
		out.MaxConnsPerHost = over.MaxConnsPerHost
	}
	if over.IdleConnTimeout != "" {
		out.IdleConnTimeout = over.IdleConnTimeout
	}
	if over.ExpectContinueTimeout != "" {
		out.ExpectContinueTimeout = over.ExpectContinueTimeout
	}
	return out
}

// mergeTransport applies defaults, then global config from file, then per-route overrides.
func mergeTransport(global, route TransportConfig) TransportConfig {
	base := defaultTransportConfig()
	out := mergeTwo(base, global)
	return mergeTwo(out, route)
}

// NewHTTPTransport builds an *http.Transport from a fully merged TransportConfig.
func NewHTTPTransport(cfg TransportConfig) (*http.Transport, error) {
	dialTimeout, err := time.ParseDuration(cfg.DialTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial_timeout: %w", err)
	}
	keepAlive, err := time.ParseDuration(cfg.KeepAlive)
	if err != nil {
		return nil, fmt.Errorf("keep_alive: %w", err)
	}
	idleConnTimeout, err := time.ParseDuration(cfg.IdleConnTimeout)
	if err != nil {
		return nil, fmt.Errorf("idle_conn_timeout: %w", err)
	}
	expectContinue, err := time.ParseDuration(cfg.ExpectContinueTimeout)
	if err != nil {
		return nil, fmt.Errorf("expect_continue_timeout: %w", err)
	}

	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: keepAlive,
		}).DialContext,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		ExpectContinueTimeout: expectContinue,
	}, nil
}
