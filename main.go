package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/IsaacDSC/proxy/internal/config"
	"github.com/IsaacDSC/proxy/internal/proxy"
	"github.com/IsaacDSC/proxy/internal/router"
	"github.com/gin-gonic/gin"
)

func main() {
	startTime := time.Now()

	configPath := flag.String("config", "config.json", "path to proxy config json")
	listenAddr := flag.String("listen", ":8080", "address to listen on")
	flag.Parse()

	compiled, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	matcher := router.NewMatcher(compiled.Routes)

	r.NoRoute(func(c *gin.Context) {
		route := matcher.MatchRoute(c.Request)
		if route == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "route not found"})
			return
		}

		if err := proxy.Forward(route, c.Writer, c.Request); err != nil {
			status := http.StatusBadGateway
			if errors.Is(err, context.DeadlineExceeded) {
				status = http.StatusGatewayTimeout
			}
			c.JSON(status, gin.H{"error": "proxy forward failed", "details": err.Error()})
		}
	})

	srv := &http.Server{
		Addr:    *listenAddr,
		Handler: r,
	}

	go func() {
		log.Printf("proxy listening on %s (started in %s)", *listenAddr, time.Since(startTime).Round(time.Millisecond))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped")
}
