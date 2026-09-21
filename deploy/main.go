// Command laya-deploy is a Go deployment server that hosts the Laya model API.
//
// It embeds System 1 decision handlers, ops endpoints, a web admin dashboard,
// metrics, hot reload, Windows service mode, and optional GitHub auto-update.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/neko233-com/laya-go/internal/engine"
	"github.com/neko233-com/laya-go/internal/httpserver"
	"github.com/neko233-com/laya-go/internal/metrics"
	"github.com/neko233-com/laya-go/internal/version"
)

//go:embed web/admin.html
var adminHTML []byte

type deployStatus struct {
	Status          string  `json:"status"`
	Role            string  `json:"role"`
	Version         string  `json:"version"`
	DeployVer       string  `json:"deploy_version"`
	Engine          string  `json:"engine"`
	EngineVer       string  `json:"engine_version"`
	Models          int     `json:"models"`
	Addr            string  `json:"addr"`
	PID             int     `json:"pid"`
	GoVersion       string  `json:"go_version"`
	GOOS            string  `json:"goos"`
	GOARCH          string  `json:"goarch"`
	StartedAt       string  `json:"started_at"`
	UptimeSec       float64 `json:"uptime_sec"`
	Executable      string  `json:"executable"`
	WorkDir         string  `json:"workdir"`
	ConfigPath      string  `json:"config_path"`
	ModelsOverlay   string  `json:"models_overlay,omitempty"`
	ServiceName     string  `json:"service_name,omitempty"`
	RunningAsSvc    bool    `json:"running_as_service"`
	HealthPath      string  `json:"health_path"`
	DecidePath      string  `json:"decide_path"`
	ModelsPath      string  `json:"models_path"`
	AdminPath       string  `json:"admin_path"`
	UpdateCheckPath string  `json:"update_check_path"`
	ConfigAPIPath   string  `json:"config_path_api"`
	TotalDecides    int64   `json:"total_decides"`
}

type configPatch struct {
	Addr       *string           `json:"addr,omitempty"`
	AutoUpdate *AutoUpdateConfig `json:"auto_update,omitempty"`
}

func main() {
	port := flag.Int("port", 7710, "listen port (used when addr empty and config empty)")
	addrFlag := flag.String("addr", "", "full listen address (overrides config and -port)")
	configFlag := flag.String("config", "", "config file path")
	serviceFlag := flag.Bool("service", false, "run as Windows service (also auto-detected under SCM)")
	flag.Parse()

	cfgPath := *configFlag
	if cfgPath == "" {
		cfgPath = resolveConfigPath()
	}
	cfg := loadConfig(cfgPath)

	if *addrFlag != "" {
		cfg.Addr = *addrFlag
	} else if flagUsed("port") || cfg.Addr == "" {
		cfg.Addr = fmt.Sprintf("0.0.0.0:%d", *port)
	}

	exe, _ := os.Executable()
	applyPendingOnStartup(exe)

	run := func(ctx context.Context) error {
		return runServer(ctx, cfg, exe)
	}

	if *serviceFlag || isWindowsService() {
		log.Printf("starting as Windows service name=LayaDeploy addr=%s", cfg.Addr)
		if err := runWindowsService(run); err != nil {
			log.Fatalf("service: %v", err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("run: %v", err)
	}
}

func flagUsed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func modelsOverlayPath(configPath string) string {
	if configPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(configPath), "models.json")
}

func runServer(ctx context.Context, cfg Config, exe string) error {
	started := time.Now()
	if exe == "" {
		exe, _ = os.Executable()
	}
	wd, _ := os.Getwd()
	reg := engine.NewRegistry()
	modelsPath := modelsOverlayPath(cfg.ConfigPath)
	if modelsPath != "" {
		if err := reg.ReloadFromJSON(modelsPath); err != nil {
			log.Printf("models overlay load failed: %v", err)
		}
	}

	met := metrics.New(100)
	laya := httpserver.New(reg)
	laya.SetRecorder(met)

	var cfgMu sync.RWMutex
	cur := cfg
	upd := newUpdater(cfg.ConfigPath, cur, exe)
	asSvc := isWindowsService()
	svcName := ""
	if runtime.GOOS == "windows" {
		svcName = "LayaDeploy"
	}

	if cur.AutoUpdate.Enabled {
		go upd.loop(ctx)
	} else {
		log.Printf("auto-update disabled in config")
	}

	mux := http.NewServeMux()
	mux.Handle("/health", laya.Handler())
	mux.Handle("/v1/", laya.Handler())

	statusFn := func() deployStatus {
		cfgMu.RLock()
		c := cur
		cfgMu.RUnlock()
		return deployStatus{
			Status:          "ok",
			Role:            "laya-deploy",
			Version:         version.Version,
			DeployVer:       version.Version,
			Engine:          engine.EngineName,
			EngineVer:       httpserver.Version,
			Models:          reg.Count(),
			Addr:            c.Addr,
			PID:             os.Getpid(),
			GoVersion:       runtime.Version(),
			GOOS:            runtime.GOOS,
			GOARCH:          runtime.GOARCH,
			StartedAt:       started.UTC().Format(time.RFC3339),
			UptimeSec:       time.Since(started).Seconds(),
			Executable:      exe,
			WorkDir:         wd,
			ConfigPath:      c.ConfigPath,
			ModelsOverlay:   reg.Source(),
			ServiceName:     svcName,
			RunningAsSvc:    asSvc,
			HealthPath:      "/health",
			DecidePath:      "/v1/decide",
			ModelsPath:      "/v1/models",
			AdminPath:       "/admin",
			UpdateCheckPath: "/deploy/update/check",
			ConfigAPIPath:   "/deploy/config",
			TotalDecides:    met.Total(),
		}
	}

	mux.HandleFunc("GET /deploy/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusFn())
	})

	mux.HandleFunc("GET /deploy/update/check", func(w http.ResponseWriter, r *http.Request) {
		res, err := upd.check(r.Context())
		if err != nil {
			snap, _ := upd.snapshot()
			snap.Message = err.Error()
			writeJSON(w, http.StatusBadGateway, snap)
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("POST /deploy/update/apply", func(w http.ResponseWriter, r *http.Request) {
		force := r.URL.Query().Get("force") == "1" || r.URL.Query().Get("force") == "true"
		res, err := upd.apply(r.Context(), force)
		if err != nil {
			writeErrJSON(w, http.StatusBadGateway, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("GET /deploy/update/status", func(w http.ResponseWriter, _ *http.Request) {
		snap, _ := upd.snapshot()
		writeJSON(w, http.StatusOK, snap)
	})

	mux.HandleFunc("GET /deploy/config", func(w http.ResponseWriter, _ *http.Request) {
		cfgMu.RLock()
		c := cur
		cfgMu.RUnlock()
		writeJSON(w, http.StatusOK, c)
	})

	mux.HandleFunc("POST /deploy/config", func(w http.ResponseWriter, r *http.Request) {
		var patch configPatch
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			writeErrJSON(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		cfgMu.Lock()
		next := cur
		if patch.Addr != nil && *patch.Addr != "" {
			next.Addr = *patch.Addr
		}
		if patch.AutoUpdate != nil {
			next.AutoUpdate = *patch.AutoUpdate
			if next.AutoUpdate.IntervalMin <= 0 {
				next.AutoUpdate.IntervalMin = 360
			}
			if next.AutoUpdate.Repo == "" {
				next.AutoUpdate.Repo = defaultConfig().AutoUpdate.Repo
			}
		}
		cur = next
		path := cur.ConfigPath
		cfgMu.Unlock()
		upd.setConfig(next)
		if path != "" {
			if err := saveConfig(path, next); err != nil {
				writeErrJSON(w, http.StatusInternalServerError, "save config: "+err.Error())
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"config": next,
			"note":   "auto_update toggles apply immediately; addr change requires service restart",
		})
	})

	mux.HandleFunc("POST /deploy/reload", func(w http.ResponseWriter, _ *http.Request) {
		cfgMu.Lock()
		path := cur.ConfigPath
		cfgMu.Unlock()
		fresh := loadConfig(path)
		if path != "" && fileExists(path) {
			cfgMu.Lock()
			// keep listen addr unless file has one; reload auto_update + other fields
			addr := cur.Addr
			if fresh.Addr != "" {
				addr = fresh.Addr
			}
			fresh.Addr = addr
			fresh.ConfigPath = path
			cur = fresh
			cfgMu.Unlock()
			upd.setConfig(fresh)
		}
		modelsPath := modelsOverlayPath(path)
		err := reg.ReloadFromJSON(modelsPath)
		if err != nil {
			writeErrJSON(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"models":        reg.Count(),
			"models_source": reg.Source(),
			"config_path":   path,
			"addr":          fresh.Addr,
			"auto_update":   fresh.AutoUpdate,
		})
	})

	mux.HandleFunc("GET /admin/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(adminHTML)
	})

	mux.HandleFunc("GET /admin/api/overview", func(w http.ResponseWriter, _ *http.Request) {
		updSnap, _ := upd.snapshot()
		cfgMu.RLock()
		c := cur
		cfgMu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":        statusFn(),
			"metrics":       met.Snapshot(),
			"models":        reg.List(),
			"models_source": reg.Source(),
			"config":        c,
			"update":        updSnap,
		})
	})

	mux.HandleFunc("POST /admin/api/reload", func(w http.ResponseWriter, _ *http.Request) {
		cfgMu.Lock()
		path := cur.ConfigPath
		cfgMu.Unlock()
		fresh := loadConfig(path)
		if path != "" && fileExists(path) {
			cfgMu.Lock()
			addr := cur.Addr
			if fresh.Addr != "" {
				addr = fresh.Addr
			}
			fresh.Addr = addr
			fresh.ConfigPath = path
			cur = fresh
			cfgMu.Unlock()
			upd.setConfig(fresh)
		}
		modelsPath := modelsOverlayPath(path)
		if err := reg.ReloadFromJSON(modelsPath); err != nil {
			writeErrJSON(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":        "ok",
			"models":        reg.Count(),
			"models_source": reg.Source(),
			"config_path":   path,
			"auto_update":   fresh.AutoUpdate,
		})
	})

	mux.HandleFunc("GET /deploy/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/deploy/" && r.URL.Path != "/deploy" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"service": "laya-deploy",
			"version": version.Version,
			"endpoints": []string{
				"GET /health",
				"GET /v1/models",
				"POST /v1/decide",
				"POST /v1/jev/decide",
				"POST /v1/predict",
				"POST /v1/explain",
				"GET /admin",
				"GET /admin/api/overview",
				"POST /admin/api/reload",
				"GET /deploy/status",
				"GET /deploy/config",
				"POST /deploy/config",
				"POST /deploy/reload",
				"GET /deploy/update/check",
				"GET /deploy/update/status",
				"POST /deploy/update/apply",
			},
		})
	})

	httpSrv := &http.Server{
		Addr:              cur.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("laya-deploy listening on http://%s engine=%s models=%d version=%s auto_update=%v service=%v admin=/admin",
			cur.Addr, engine.EngineName, reg.Count(), version.Version, cur.AutoUpdate.Enabled, asSvc)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("laya-deploy shutdown signal")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Println("laya-deploy stopped")
	return nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErrJSON(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
