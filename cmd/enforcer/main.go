package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/audit"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/proxy"
)

const PORT = ":8080"

func main() {
	fmt.Println("Hello World")
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

	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

}
