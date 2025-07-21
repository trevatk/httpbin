package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	http "github.com/trevatk/safehttp"
)

var (
	appVersion string = ""
	gitCommit  string = ""
	buildDate  string = ""
)

type golang struct {
	version      string
	arch         string
	os           string
	compiler     string
	numCPU       int
	numGoRoutine int
}

type who struct {
	hostname      string
	appVersion    string
	gitCommit     string
	buildDate     string
	golang        golang
	uid, gid, pid int
	extraEnvs     map[string]string
}

func newWho() (who, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return who{}, fmt.Errorf("os.Hostname: %w", err)
	}

	extraEnvs := os.Getenv("EXTRA_ENVS")
	ee := make(map[string]string)
	for _, env := range strings.Split(extraEnvs, ",") {
		value := os.Getenv(env)
		ee[env] = value
	}

	return who{
		hostname:   hostname,
		appVersion: appVersion,
		gitCommit:  gitCommit,
		buildDate:  buildDate,
		golang: golang{
			version:      runtime.Version(),
			compiler:     runtime.Compiler,
			arch:         runtime.GOARCH,
			os:           runtime.GOOS,
			numCPU:       runtime.NumCPU(),
			numGoRoutine: runtime.NumGoroutine(),
		},
		uid:       os.Getuid(),
		gid:       os.Getgid(),
		pid:       os.Getpid(),
		extraEnvs: ee,
	}, nil
}

type httpServer struct {
	logger *slog.Logger
	who    who
}

func newHttpServer(logger *slog.Logger, who who) *httpServer {
	return &httpServer{
		logger: logger,
		who:    who,
	}
}

func (hs *httpServer) health(w http.ResponseWriter, r *http.Request) {
	hs.logger.DebugContext(r.Context(), "health check")
	w.SetStatus(http.StatusOK)
	w.Write([]byte("OK"))
}

func (hs *httpServer) echo(w http.ResponseWriter, r *http.Request) {
	hs.logger.DebugContext(r.Context(), "echo")
	var respMsg string
	if msg, ok := r.Params["{msg}"]; ok {
		respMsg = msg
	}

	w.SetStatus(http.StatusAccepted)
	if respMsg == "ping" {
		// easter egg
		w.SetStatus(http.StausTeapot)
		respMsg = "pong"
	}

	_, err := w.Write([]byte(respMsg))
	if err != nil {
		hs.logger.ErrorContext(r.Context(), "w.Write", slog.String("error", err.Error()))
		w.SetStatus(http.StatusInternalServer)
		return
	}
}

func (hs *httpServer) whoami(w http.ResponseWriter, r *http.Request) {
	hs.logger.DebugContext(r.Context(), "whoami")
	resp := newWhoamiResponse(hs.who)
	w.SetStatus(http.StatusAccepted)
	w.SetHeader("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		hs.logger.ErrorContext(r.Context(), "Encode", slog.String("error", err.Error()))
		w.SetStatus(http.StatusInternalServer)
	}
}

type golangResponse struct {
	Arch         string `json:"arch"`
	OS           string `json:"os"`
	NumCPU       int    `json:"num_cpu"`
	NumGoRoutine int    `json:"num_go_routine"`
	Version      string `json:"version"`
	Compiler     string `json:"compiler"`
}

type whoamiResponse struct {
	Hostname   string            `json:"hostname"`
	AppVersion string            `json:"app_version"`
	GitCommit  string            `json:"git_commit"`
	BuildDate  string            `json:"build_date"`
	Golang     golangResponse    `json:"go"`
	UID        int               `json:"uid"`
	GID        int               `json:"gid"`
	PID        int               `json:"pid"`
	ExtraEnvs  map[string]string `json:"extra_envs"`
}

func newWhoamiResponse(who who) *whoamiResponse {
	return &whoamiResponse{
		Hostname:   who.hostname,
		AppVersion: who.appVersion,
		GitCommit:  who.gitCommit,
		BuildDate:  who.buildDate,
		Golang: golangResponse{
			Arch:         who.golang.arch,
			OS:           who.golang.os,
			Version:      who.golang.version,
			Compiler:     who.golang.compiler,
			NumCPU:       who.golang.numCPU,
			NumGoRoutine: who.golang.numGoRoutine,
		},
		UID:       who.uid,
		GID:       who.gid,
		PID:       who.pid,
		ExtraEnvs: who.extraEnvs,
	}
}

func newRouter(hs *httpServer) *http.Mux {
	mux := http.NewServeMux()

	mux.Get("/health", hs.health)
	mux.Get("/echo/{msg}", hs.echo)
	mux.Get("/whoami", hs.whoami)

	return mux
}

func newLogger(level string) *slog.Logger {
	var ll slog.Level
	switch strings.ToLower(level) {
	case "warn":
		ll = slog.LevelWarn
	case "info":
		ll = slog.LevelInfo
	case "error":
		ll = slog.LevelError
	default:
		ll = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: ll,
	}))
}

func main() {
	logger := newLogger(os.Getenv("LOG_LEVEL"))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	whoami, err := newWho()
	if err != nil {
		logger.ErrorContext(ctx, "build whoami", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.InfoContext(ctx, "service configuration",
		slog.String("hostname", whoami.hostname),
		slog.String("go_version", whoami.golang.version),
		slog.Int("UID", whoami.uid),
		slog.Int("GID", whoami.gid),
		slog.Int("PID", whoami.pid),
		slog.Any("etra_envs", whoami.extraEnvs),
	)

	hs := newHttpServer(logger, whoami)
	r := newRouter(hs)

	serverAddr := os.Getenv("SERVER_ADDR")
	if serverAddr == "" {
		serverAddr = ":8080"
	}

	serverOpts := []http.ServerOption{
		http.WithAddr(serverAddr),
		http.WithMux(r),
	}

	logger.InfoContext(ctx, "start http/1 server", slog.String("server_addr", serverAddr))
	s := http.NewServer(serverOpts...)
	s.Serve(ctx)
}
