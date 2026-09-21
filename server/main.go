// Command laya-server runs the open Laya System 1 decision server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/neko233-com/laya-go/internal/engine"
	"github.com/neko233-com/laya-go/internal/httpserver"
)

func main() {
	addr := flag.String("addr", envOrDefault("LAYA_ADDR", "0.0.0.0:7710"), "listen address")
	flag.Parse()

	srv := httpserver.New(engine.NewRegistry())
	httpSrv := &http.Server{
		Addr:              httpserver.AddrLabel(*addr),
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("laya-server listening on http://%s engine=%s version=%s", httpSrv.Addr, engine.EngineName, httpserver.Version)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
