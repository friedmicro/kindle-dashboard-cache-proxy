package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

type config struct {
	upstream string
	outFile  string
	interval time.Duration
	listen   string
}

func parseConfig() config {
	var (
		upstream = envOrDefault("UPSTREAM", "http://kindle-live.dashboard.friedmicro-lab.org/")
		outFile  = envOrDefault("OUT_FILE", "/tmp/dashboard.png")
		listen   = envOrDefault("LISTEN_ADDR", ":8090")
	)

	intervalSec := envOrDefaultInt("REFRESH_INTERVAL_SEC", 60)

	return config{
		upstream: upstream,
		outFile:  outFile,
		interval: time.Duration(intervalSec) * time.Second,
		listen:   listen,
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func newUpstreamClient(interval time.Duration) *http.Client {
	timeout := interval / 2
	if timeout > 30*time.Second {
		timeout = 30 * time.Second
	}
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func fetch(upstream, outFile string, client *http.Client) error {
	resp, err := client.Get(upstream)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &upstreamError{code: resp.StatusCode}
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	dir := filepath.Dir(outFile)
	tmp := outFile + ".tmp"
	f, err := os.CreateTemp(dir, filepath.Base(tmp)+".*")
	if err != nil {
		return err
	}
	tmpName := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmpName)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpName)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, outFile)
}

func refreshLoop(cfg config, client *http.Client, stop <-chan struct{}) {
	fetchNow := func() {
		if err := fetch(cfg.upstream, cfg.outFile, client); err != nil {
			log.Printf("refresh failed (keeping previous image): %v", err)
			return
		}
		log.Printf("refreshed %s from %s", cfg.outFile, cfg.upstream)
	}

	fetchNow()
	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fetchNow()
		case <-stop:
			return
		}
	}
}

func serveImage(outFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(outFile)
		if err != nil {
			http.Error(w, "dashboard image not available yet", http.StatusServiceUnavailable)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, filepath.Base(outFile), time.Now(), f)
	}
}

type upstreamError struct{ code int }

func (e *upstreamError) Error() string {
	return "upstream returned HTTP " + http.StatusText(e.code)
}

func main() {
	cfg := parseConfig()
	client := newUpstreamClient(cfg.interval)

	stop := make(chan struct{})
	go refreshLoop(cfg, client, stop)

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveImage(cfg.outFile))
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{Addr: cfg.listen, Handler: mux}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		close(stop)
		log.Println("shutting down")
		srv.Close()
	}()

	log.Printf("cache-proxy serving %s on %s (refresh every %s)", cfg.outFile, cfg.listen, cfg.interval)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
