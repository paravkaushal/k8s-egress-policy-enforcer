package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
)

func main() {
	fmt.Println("Hello World")
	policyFile := flag.String("policy-file", "policies/example.yaml", "path to the policy YAML/JSON file")
	flag.Parse()
	cfg, err := policy.Load(*policyFile)
	if err != nil {
		slog.Error("failed to load policy file", "error", err)
		os.Exit(1)
	}

	_ = policy.NewEngine(cfg)
}
