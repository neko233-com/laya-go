// Command laya-deploy is a Go deployment server that hosts the Laya model API.
//
// It embeds the same System 1 decision handlers as laya-server and adds
// deployment status endpoints for local/ops use.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/neko233-com/laya-go/internal/engine"
	"github.com/neko233-com/laya-go/internal/httpserver"
)

const deployVersion = "0.1.0"

type deployStatus struct {
	Status     string  `json:"status"`
	Role       string  `json:"role"`
	DeployVer  string  `json:"deploy_version"`
	Engine     string  `json:"engine"`
	EngineVer  string  `json:"engine_version"`
	Models     int     `json:"models"`
	Addr       string  `json:"addr"`
	PID        int     `json:"pid"`
	GoVersion  string  `json:"go_version"`
	GOOS       string  `json:"goos"`
	GOARCH     string  `json:"goarch"`
	StartedAt  string  `json:"started_at"`
	UptimeSec  float64 `json:"uptime_sec"`
	Executable string  `json:"executable"`
	WorkDir    string  `json:"workdir"`
	HealthPath string  `json:"health_path"`
	DecidePath string  `json:"decide_path"`
	ModelsPath string  `json:"models_path"`
}

func main() {
	port := flag.Int("port", 7400, "listen port for deploy server + Laya model service")
	addrFlag := flag.String("addr", "", "full listen address (overrides -port)")
	flag.Parse()

	addr := *addrFlag
	if addr == "" {
		addr = fmt.Sprintf("127.0.0.1:%d", *port)
	}

	started := time.Now()
	exe, _ := os.Executable()
	wd, _ := os.Getwd()
	reg := engine.NewRegistry()
	laya := httpserver.New(reg)
	mux := http.NewServeMux()

	// Mount full Laya model service under the deploy server.
	mux.Handle("/health", laya.Handler())
	mux.Handle("/v1/", laya.Handler())
	mux.HandleFunc("GET /deploy/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, deployStatus{
			Status:     "ok",
			Role:       "laya-deploy",
			DeployVer:  deployVersion,
			Engine:     engine.EngineName,
			EngineVer:  httpserver.Version,
			Models:     reg.Count(),
			Addr:       addr,
			PID:        os.Getpid(),
			GoVersion:  runtime.Version(),
			GOOS:       runtime.GOOS,
			GOARCH:     runtime.GOARCH,
			StartedAt:  started.UTC().Format(time.RFC3339),
			UptimeSec:  time.Since(started).Seconds(),
			Executable: exe,
			WorkDir:    wd,
			HealthPath: "/health",
			DecidePath: "/v1/decide",
			ModelsPath: "/v1/models",
		})
	})
	mux.HandleFunc("GET /deploy/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deploy/" && r.URL.Path != "/deploy" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"service": "laya-deploy",
			"endpoints": []string{
				"GET /health",
				"GET /v1/models",
				"POST /v1/decide",
				"POST /v1/jev/decide",
				"POST /v1/predict",
				"POST /v1/explain",
				"GET /deploy/status",
			},
		})
	})

	httpSrv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("laya-deploy listening on http://%s (port %d) engine=%s models=%d", addr, *port, engine.EngineName, reg.Count())
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
	log.Println("laya-deploy stopped")
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
