package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/IsaacDSC/proxy/internal/config"
	"github.com/IsaacDSC/proxy/internal/proxy"
	"github.com/IsaacDSC/proxy/internal/router"
	"github.com/gin-gonic/gin"
)

func main() {
	configPath := flag.String("config", "config.json", "path to proxy config json")
	listenAddr := flag.String("listen", ":8080", "address to listen on")
	flag.Parse()

	compiled, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	matcher := router.NewMatcher(compiled.Routes)

	r.NoRoute(func(c *gin.Context) {
		route := matcher.MatchRoute(c.Request.Method, c.Request.URL.Path, c.Request.Header)
		if route == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		rewriteMethod, rewritePath := route.ResolveRewrite(c.Request.Method, c.Request.URL.Path)
		if err := proxy.ForwardWithRewrite(client, route.Target, rewriteMethod, rewritePath, c.Writer, c.Request); err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, context.DeadlineExceeded) {
				status = http.StatusGatewayTimeout
			}
			c.JSON(status, gin.H{"error": "proxy forward failed", "details": err.Error()})
		}
	})

	log.Printf("proxy listening on %s", *listenAddr)
	if err := r.Run(*listenAddr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
