package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/audit"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/proxy"
)

const PORT = ":8080"

func main() {
	policyFile := flag.String("policy-file", "policies/example.yaml", "path to the policy YAML/JSON file")
	flag.Parse()
	cfg, err := policy.Load(*policyFile)
	if err != nil {
		slog.Error("failed to load policy file", "error", err)
		os.Exit(1)
	}

	engine := policy.NewEngine(cfg)
	auditLogger := audit.New(os.Stdout)
	handler := proxy.NewHandler(engine, auditLogger)

	mux := http.NewServeMux()
	mux.Handle("/proxy", handler)

	srv := &http.Server{
		Addr:    PORT,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("enforcer listening", "addr", PORT, "policy_file", *policyFile, "policy_count", len(cfg.Policies))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}

}
