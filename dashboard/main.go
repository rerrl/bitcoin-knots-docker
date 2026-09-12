// Dashboard for Bitcoin Knots + Fulcrum Docker suite.
//
// A lightweight, dependency-free HTTP server (stdlib only) that observes the
// stack and exposes a small read-only JSON API consumed by a single embedded
// HTML page. It performs no writes to the host and never touches the docker
// socket. It reads:
//
//   - Bitcoin sync state  -> bitcoind JSON-RPC via the .cookie file (same
//     approach Fulcrum already uses), plus bitcoind's own debug.log.
//   - Fulcrum reachability-> a lightweight TCP banner read on the Electrum port.
//   - Tor status          -> the hidden service hostname file.
//   - Logs                -> files in a shared ./logs volume (fulcrum, tor)
//                            and ./bitcoin-data/debug.log (bitcoind).
//   - Configs             -> the active bitcoin.conf (from the datadir) and the
//                            fulcrum.conf baked into the image.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*.html
var webFS embed.FS

// config holds the runtime settings, sourced from a baked-in config file with
// env-overrides for the paths that vary per deployment (used by compose).
type config struct {
	Listen        string // e.g. "0.0.0.0:8080"
	BitcoinRPC    string // e.g. "http://bitcoind:8332"
	BitcoinDir    string // dir containing .cookie and debug.log, e.g. "/home/bitcoin/.bitcoin"
	FulcrumHost   string // e.g. "fulcrum"
	FulcrumPort   string // e.g. "50001"
	LogDir        string // e.g. "/logs"
	FulcrumConf   string // e.g. "/confs/fulcrum.conf"
	TorHostname   string // path to the onion hostname file
	BitcoinConf   string // path to the active bitcoin.conf
	LogTailLines  int
	LogTailBytes  int64
}

func defaultConfig() config {
	return config{
		Listen:       envOr("DASHBOARD_LISTEN", "0.0.0.0:8080"),
		BitcoinRPC:   envOr("DASHBOARD_BITCOIND_RPC", "http://bitcoind:8332"),
		BitcoinDir:   envOr("DASHBOARD_BITCOIND_DATADIR", "/home/bitcoin/.bitcoin"),
		FulcrumHost:  envOr("DASHBOARD_FULCRUM_HOST", "fulcrum"),
		FulcrumPort:  envOr("DASHBOARD_FULCRUM_PORT", "50001"),
		LogDir:       envOr("DASHBOARD_LOGDIR", "/logs"),
		FulcrumConf:  envOr("DASHBOARD_FULCRUM_CONF", "/confs/fulcrum.conf"),
		TorHostname:  envOr("DASHBOARD_TOR_HOSTNAME", "/tor-data/hostname"),
		BitcoinConf:  envOr("DASHBOARD_BITCOIN_CONF", "/home/bitcoin/.bitcoin/bitcoin.conf"),
		LogTailLines: 200,
		LogTailBytes: 256 * 1024,
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

// ---------------------------------------------------------------------------
// Bitcoin RPC
// ---------------------------------------------------------------------------

type bitcoinClient struct {
	rc     *http.Client
	rpcURL string
	cookie string // decoded "<user>:<password>"
}

func newBitcoinClient(url, datadir string) *bitcoinClient {
	c := &bitcoinClient{
		rc:     &http.Client{Timeout: 5 * time.Second},
		rpcURL: url,
	}
	// The cookie file contains "<user>:<password>" on a single line.
	if cookie, err := os.ReadFile(datadir + "/.cookie"); err == nil {
		c.cookie = strings.TrimSpace(string(cookie))
	}
	return c
}

func (b *bitcoinClient) rpc(method string, params []any) (map[string]any, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "1.0",
		"id":      "dashboard",
		"method":  method,
		"params":  params,
	})
	req, err := http.NewRequest(http.MethodPost, b.rpcURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if b.cookie != "" {
		user, pass, _ := strings.Cut(b.cookie, ":")
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":"+pass)))
	}
	resp, err := b.rc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Result map[string]any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("rpc %s: %s (code %d)", method, parsed.Error.Message, parsed.Error.Code)
	}
	return parsed.Result, nil
}

func (b *bitcoinClient) status() (map[string]any, bool) {
	res, err := b.rpc("getblockchaininfo", nil)
	if err != nil {
		return map[string]any{"error": err.Error()}, false
	}
	return res, true
}

// ---------------------------------------------------------------------------
// Fulcrum reachability
// ---------------------------------------------------------------------------

func fulcrumReachable(host, port string) map[string]any {
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return map[string]any{"reachable": false, "error": err.Error()}
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	// Fulcrum (electrum protocol) sends a banner line on connect.
	line, _ := readLine(conn)
	return map[string]any{"reachable": true, "banner": line}
}

func readLine(r io.Reader) (string, error) {
	buf := make([]byte, 0, 256)
	tmp := make([]byte, 1)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			if tmp[0] == '\n' {
				return strings.TrimSpace(string(buf)), nil
			}
			buf = append(buf, tmp[0])
			if len(buf) > 4096 {
				return strings.TrimSpace(string(buf)), nil
			}
		}
		if err != nil {
			return strings.TrimSpace(string(buf)), err
		}
	}
}

// ---------------------------------------------------------------------------
// Tor
// ---------------------------------------------------------------------------

func torStatus(hostnamePath string) map[string]any {
	data, err := os.ReadFile(hostnamePath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"enabled": false, "onion": "", "error": "tor not enabled (no hidden service hostname)"}
		}
		return map[string]any{"enabled": true, "onion": "", "error": err.Error()}
	}
	onion := strings.TrimSpace(string(data))
	return map[string]any{"enabled": onion != "", "onion": onion}
}

// ---------------------------------------------------------------------------
// Logs / configs
// ---------------------------------------------------------------------------

// tailFile returns the last maxLines lines of a file by only reading the final
// maxBytes window, so multi-hundred-MB debug logs stay cheap.
func tailFile(path string, maxLines int, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	start := fi.Size() - maxBytes
	if start < 0 {
		start = 0
	}
	if start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return "", err
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
}

func readFileOr(path, errText string) (string, string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errText
	}
	return string(data), ""
}

// ---------------------------------------------------------------------------
// HTTP API
// ---------------------------------------------------------------------------

type server struct {
	cfg     config
	bitcoin *bitcoinClient
}

func newServer(cfg config) *server {
	return &server{
		cfg:     cfg,
		bitcoin: newBitcoinClient(cfg.BitcoinRPC, cfg.BitcoinDir),
	}
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/logs", s.handleLogs)
	mux.HandleFunc("GET /api/confs", s.handleConfs)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
	})
	return mux
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := webFS.ReadFile("web/index.html")
	if err != nil {
		http.Error(w, "index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var (
		bc  map[string]any
		bok bool
	)
	if err := runCtx(ctx, func() {
		bc, bok = s.bitcoin.status()
	}); err != nil {
		bc = map[string]any{"error": err.Error()}
		bok = false
	}

	// Fulcrum and Tor checks are quick local reads/dials.
	fulcrum := fulcrumReachable(s.cfg.FulcrumHost, s.cfg.FulcrumPort)
	tor := torStatus(s.cfg.TorHostname)

	writeJSON(w, map[string]any{
		"bitcoin": bc,
		"up":      bok,
		"fulcrum": fulcrum,
		"tor":     tor,
		"sparrow": map[string]any{
			"server": tor["onion"],
			"port":   "50001",
			"protocol": "TCP (SSL disabled)",
		},
	})
}

func runCtx(ctx context.Context, fn func()) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *server) handleLogs(w http.ResponseWriter, r *http.Request) {
	service := strings.ToLower(r.URL.Query().Get("service"))
	maxLines, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	if maxLines <= 0 || maxLines > 2000 {
		maxLines = s.cfg.LogTailLines
	}

	var path string
	switch service {
	case "bitcoind":
		path = s.cfg.BitcoinDir + "/debug.log"
	case "fulcrum":
		path = s.cfg.LogDir + "/fulcrum.log"
	case "tor":
		path = s.cfg.LogDir + "/tor.log"
	default:
		writeJSON(w, map[string]any{"error": "unknown service; use bitcoind, fulcrum, or tor"})
		return
	}

	content, err := tailFile(path, maxLines, s.cfg.LogTailBytes)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, map[string]any{"service": service, "content": "", "error": "log file not available yet"})
			return
		}
		writeJSON(w, map[string]any{"service": service, "content": "", "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"service": service, "content": content})
}

func (s *server) handleConfs(w http.ResponseWriter, r *http.Request) {
	bitcoinConf, bcErr := readFileOr(s.cfg.BitcoinConf, "bitcoin.conf not found in datadir")
	fulcrumConf, fcErr := readFileOr(s.cfg.FulcrumConf, "fulcrum.conf not found")
	writeJSON(w, map[string]any{
		"bitcoin": map[string]any{"content": bitcoinConf, "error": bcErr},
		"fulcrum": map[string]any{"content": fulcrumConf, "error": fcErr},
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func main() {
	cfg := defaultConfig()
	srv := newServer(cfg)

	httpSrv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	addr := cfg.Listen
	if strings.HasPrefix(addr, "0.0.0.0") {
		addr = "127.0.0.1" + strings.TrimPrefix(addr, "0.0.0.0")
	}
	fmt.Printf("dashboard listening on %s (reach via http://localhost of host)\n", cfg.Listen)
	fmt.Printf("direct: http://%s\n", addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "dashboard: %v\n", err)
		os.Exit(1)
	}
}