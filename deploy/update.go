package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/neko233-com/laya-go/internal/version"
)

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type releaseInfo struct {
	TagName    string         `json:"tag_name"`
	Name       string         `json:"name"`
	Prerelease bool           `json:"prerelease"`
	Draft      bool           `json:"draft"`
	HTMLURL    string         `json:"html_url"`
	Published  string         `json:"published_at"`
	Assets     []releaseAsset `json:"assets"`
}

type updateCheckResult struct {
	CurrentVersion  string   `json:"current_version"`
	LatestVersion   string   `json:"latest_version"`
	TagName         string   `json:"tag_name"`
	ReleaseURL      string   `json:"release_url"`
	PublishedAt     string   `json:"published_at"`
	UpdateAvailable bool     `json:"update_available"`
	AutoUpdateOn    bool     `json:"auto_update_enabled"`
	AutoApplyOn     bool     `json:"auto_apply_enabled"`
	Repo            string   `json:"repo"`
	MatchedAssets   []string `json:"matched_assets"`
	CheckedAt       string   `json:"checked_at"`
	Message         string   `json:"message,omitempty"`
}

type updater struct {
	mu          sync.Mutex
	cfgPath     string
	cfg         Config
	exePath     string
	client      *http.Client
	lastCheck   *updateCheckResult
	lastCheckAt time.Time
	lastError   string
	restartFn   func()
	serviceName string
}

func newUpdater(cfgPath string, cfg Config, exePath string) *updater {
	u := &updater{
		cfgPath:     cfgPath,
		cfg:         cfg,
		exePath:     exePath,
		client:      &http.Client{Timeout: 30 * time.Second},
		serviceName: windowsServiceName(),
	}
	u.restartFn = u.restartServiceOrProcess
	return u
}

func windowsServiceName() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	return "LayaDeploy"
}

func (u *updater) setConfig(cfg Config) {
	u.mu.Lock()
	u.cfg = cfg
	u.mu.Unlock()
}

func (u *updater) currentConfig() Config {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.cfg
}

func (u *updater) snapshot() (updateCheckResult, string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	res := updateCheckResult{
		CurrentVersion: version.Version,
		AutoUpdateOn:   u.cfg.AutoUpdate.Enabled,
		AutoApplyOn:    u.cfg.AutoUpdate.AutoApply,
		Repo:           u.cfg.AutoUpdate.Repo,
		CheckedAt:      u.lastCheckAt.UTC().Format(time.RFC3339),
		Message:        u.lastError,
	}
	if u.lastCheck != nil {
		cp := *u.lastCheck
		cp.CurrentVersion = version.Version
		cp.AutoUpdateOn = u.cfg.AutoUpdate.Enabled
		cp.AutoApplyOn = u.cfg.AutoUpdate.AutoApply
		cp.Repo = u.cfg.AutoUpdate.Repo
		cp.CheckedAt = u.lastCheckAt.UTC().Format(time.RFC3339)
		if u.lastError != "" {
			cp.Message = u.lastError
		}
		return cp, u.lastError
	}
	return res, u.lastError
}

func (u *updater) check(ctx context.Context) (*updateCheckResult, error) {
	cfg := u.currentConfig()
	repo := cfg.AutoUpdate.Repo
	if repo == "" {
		repo = "neko233-com/laya-go"
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "laya-deploy/"+version.Version)
	resp, err := u.client.Do(req)
	if err != nil {
		u.noteError(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		u.noteError(err.Error())
		return nil, err
	}
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("github http %d: %s", resp.StatusCode, truncate(string(body), 200))
		u.noteError(msg)
		return nil, fmt.Errorf("%s", msg)
	}
	var releases []releaseInfo
	if err := json.Unmarshal(body, &releases); err != nil {
		u.noteError(err.Error())
		return nil, err
	}
	latest := pickRelease(releases, cfg.AutoUpdate.Prerelease)
	if latest == nil {
		res := &updateCheckResult{
			CurrentVersion: version.Version,
			AutoUpdateOn:   cfg.AutoUpdate.Enabled,
			AutoApplyOn:    cfg.AutoUpdate.AutoApply,
			Repo:           repo,
			Message:        "no published release found",
			CheckedAt:      time.Now().UTC().Format(time.RFC3339),
		}
		u.noteResult(res)
		return res, nil
	}
	latestVer := normalizeTag(latest.TagName)
	matched := matchAssets(*latest, runtime.GOOS, runtime.GOARCH)
	res := &updateCheckResult{
		CurrentVersion:  version.Version,
		LatestVersion:   latestVer,
		TagName:         latest.TagName,
		ReleaseURL:      latest.HTMLURL,
		PublishedAt:     latest.Published,
		UpdateAvailable: compareVersions(latestVer, version.Version) > 0,
		AutoUpdateOn:    cfg.AutoUpdate.Enabled,
		AutoApplyOn:     cfg.AutoUpdate.AutoApply,
		Repo:            repo,
		MatchedAssets:   matched,
		CheckedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	if !res.UpdateAvailable {
		res.Message = "already on latest published release"
	}
	u.noteResult(res)
	return res, nil
}

func (u *updater) noteResult(res *updateCheckResult) {
	u.mu.Lock()
	u.lastCheck = res
	u.lastCheckAt = time.Now()
	u.lastError = ""
	u.mu.Unlock()
	log.Printf("update-check current=%s latest=%s available=%v", res.CurrentVersion, res.LatestVersion, res.UpdateAvailable)
}

func (u *updater) noteError(msg string) {
	u.mu.Lock()
	u.lastError = msg
	u.lastCheckAt = time.Now()
	u.mu.Unlock()
	log.Printf("update-check error: %s", msg)
}

func (u *updater) apply(ctx context.Context, force bool) (*updateCheckResult, error) {
	res, err := u.check(ctx)
	if err != nil {
		return nil, err
	}
	if !res.UpdateAvailable && !force {
		res.Message = "no update to apply"
		return res, nil
	}
	if len(res.MatchedAssets) == 0 {
		res.Message = "update tag found but no matching release assets for this platform; publish assets via deploy/publish-release.ps1"
		return res, nil
	}
	assetName := preferDeployAsset(res.MatchedAssets)
	assetURL := ""
	cfg := u.currentConfig()
	// Re-fetch asset URL from stored check is insufficient; call GitHub again for asset list.
	rel, err := u.fetchReleaseByTag(ctx, cfg.AutoUpdate.Repo, res.TagName)
	if err != nil {
		u.noteError(err.Error())
		return nil, err
	}
	for _, a := range rel.Assets {
		if a.Name == assetName {
			assetURL = a.BrowserDownloadURL
			break
		}
	}
	if assetURL == "" {
		msg := fmt.Sprintf("asset %s missing on release %s", assetName, res.TagName)
		u.noteError(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	exe := u.exePath
	if exe == "" {
		exe, _ = os.Executable()
	}
	dir := filepath.Dir(exe)
	pending := exe + ".new"
	if err := u.downloadFile(ctx, assetURL, pending); err != nil {
		u.noteError(err.Error())
		return nil, err
	}
	// Windows allows renaming a running image; swap layout for next process start.
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		// If rename fails because path missing, still try copy-style.
		log.Printf("rename current exe to .old failed: %v", err)
	}
	if err := os.Rename(pending, exe); err != nil {
		// try copy fallback
		if cerr := copyFile(pending, exe); cerr != nil {
			u.noteError(cerr.Error())
			return nil, cerr
		}
		_ = os.Remove(pending)
	}
	// Also refresh CLI/MCP if present in same directory.
	u.downloadSiblingAssets(ctx, cfg.AutoUpdate.Repo, res.TagName, dir)

	res.Message = fmt.Sprintf("staged %s; restarting to activate", assetName)
	u.noteResult(res)
	log.Printf("update applied staging complete: %s -> %s", res.TagName, exe)

	if u.restartFn != nil {
		go func() {
			time.Sleep(400 * time.Millisecond)
			u.restartFn()
		}()
	}
	return res, nil
}

func (u *updater) downloadSiblingAssets(ctx context.Context, repo, tag, dir string) {
	if repo == "" || tag == "" || dir == "" {
		return
	}
	rel, err := u.fetchReleaseByTag(ctx, repo, tag)
	if err != nil {
		return
	}
	want := map[string]string{
		"laya":     filepath.Join(dir, "laya"+exeSuffix()),
		"laya-mcp": filepath.Join(dir, "laya-mcp"+exeSuffix()),
	}
	matched := matchAssets(*rel, runtime.GOOS, runtime.GOARCH)
	for _, name := range matched {
		for prefix, dest := range want {
			if strings.HasPrefix(strings.ToLower(name), prefix) {
				var url string
				for _, a := range rel.Assets {
					if a.Name == name {
						url = a.BrowserDownloadURL
						break
					}
				}
				if url == "" {
					continue
				}
				tmp := dest + ".new"
				if err := u.downloadFile(ctx, url, tmp); err != nil {
					log.Printf("update sibling %s failed: %v", name, err)
					continue
				}
				_ = os.Rename(dest, dest+".old")
				if err := os.Rename(tmp, dest); err != nil {
					if cerr := copyFile(tmp, dest); cerr == nil {
						_ = os.Remove(tmp)
					}
				}
				log.Printf("updated sibling binary %s", dest)
			}
		}
	}
}

func (u *updater) fetchReleaseByTag(ctx context.Context, repo, tag string) (*releaseInfo, error) {
	if repo == "" {
		repo = "neko233-com/laya-go"
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "laya-deploy/"+version.Version)
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("github http %d for tag %s", resp.StatusCode, tag)
	}
	var rel releaseInfo
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func (u *updater) downloadFile(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "laya-deploy/"+version.Version)
	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download %s: http %d", url, resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, io.LimitReader(resp.Body, 256<<20)); err != nil {
		return err
	}
	return f.Close()
}

func (u *updater) restartServiceOrProcess() {
	if runtime.GOOS == "windows" && u.serviceName != "" && isWindowsService() {
		log.Printf("restarting windows service %s to activate update", u.serviceName)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			fmt.Sprintf("Restart-Service -Name %s -Force", u.serviceName))
		_ = cmd.Start()
		return
	}
	log.Printf("update staged; restart laya-deploy manually to activate")
}

func (u *updater) loop(ctx context.Context) {
	// Startup check always runs when enabled; interval repeats.
	u.tick(ctx)
	cfg := u.currentConfig()
	mins := cfg.AutoUpdate.IntervalMin
	if mins <= 0 {
		mins = 360
	}
	t := time.NewTicker(time.Duration(mins) * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			u.tick(ctx)
		}
	}
}

func (u *updater) tick(ctx context.Context) {
	cfg := u.currentConfig()
	if !cfg.AutoUpdate.Enabled {
		log.Printf("auto-update disabled; skip check")
		return
	}
	res, err := u.check(ctx)
	if err != nil {
		return
	}
	if res.UpdateAvailable && cfg.AutoUpdate.AutoApply {
		if _, err := u.apply(ctx, false); err != nil {
			log.Printf("auto-apply failed: %v", err)
		}
	}
}

func applyPendingOnStartup(exe string) {
	if exe == "" {
		return
	}
	pending := exe + ".new"
	if !fileExists(pending) {
		return
	}
	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err == nil {
		if err := os.Rename(pending, exe); err != nil {
			log.Printf("startup pending apply failed: %v", err)
			return
		}
		log.Printf("startup: staged pending update into place; current process still old image until restart")
		return
	}
	// exe may not exist yet
	if err := os.Rename(pending, exe); err != nil {
		log.Printf("startup pending rename failed: %v", err)
	}
}

func pickRelease(all []releaseInfo, allowPre bool) *releaseInfo {
	for i := range all {
		r := all[i]
		if r.Draft {
			continue
		}
		if r.Prerelease && !allowPre {
			continue
		}
		if strings.TrimSpace(r.TagName) == "" {
			continue
		}
		return &r
	}
	return nil
}

func normalizeTag(tag string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(tag), "v"))
}

// compareVersions returns 1 if a>b, -1 if a<b, 0 if equal/unknown.
func compareVersions(a, b string) int {
	pa := parseSemver(a)
	pb := parseSemver(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return 1
		}
		if pa[i] < pb[i] {
			return -1
		}
	}
	return 0
}

func parseSemver(v string) [3]int {
	v = normalizeTag(v)
	// strip pre-release/build
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	var out [3]int
	parts := strings.Split(v, ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			// non-numeric tail: ignore
			continue
		}
		out[i] = n
	}
	return out
}

func matchAssets(rel releaseInfo, goos, goarch string) []string {
	var out []string
	for _, a := range rel.Assets {
		n := strings.ToLower(a.Name)
		if !strings.Contains(n, goos) && !strings.Contains(n, platformAlias(goos)) {
			// allow generic amd64-only names for windows if they include exe and arch
			if !(goos == "windows" && strings.HasSuffix(n, ".exe") && strings.Contains(n, goarch)) {
				continue
			}
		}
		if !strings.Contains(n, goarch) && !strings.Contains(n, archAlias(goarch)) {
			continue
		}
		if goos == "windows" && !strings.HasSuffix(n, ".exe") {
			continue
		}
		out = append(out, a.Name)
	}
	return out
}

func platformAlias(goos string) string {
	if goos == "windows" {
		return "win"
	}
	return goos
}

func archAlias(arch string) string {
	if arch == "amd64" {
		return "x86_64"
	}
	return arch
}

func preferDeployAsset(names []string) string {
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), "laya-deploy") {
			return n
		}
	}
	return names[0]
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmpcopy"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
