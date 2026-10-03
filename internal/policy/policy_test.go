package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
)

func writeTempPolicyFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "policies.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing temp policy file: %v", err)
	}
	return path
}

func TestLoad_ValidFile(t *testing.T) {
	path := writeTempPolicyFile(t, `
policies:
  - workload: payment-service
    destination: api.github.com
    methods: [GET]
    effect: allow
  - workload: payment-service
    destination: "*"
    effect: deny
`)

	cfg, err := policy.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(cfg.Policies))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := policy.Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestLoad_InvalidEffect(t *testing.T) {
	path := writeTempPolicyFile(t, `
policies:
  - workload: payment-service
    destination: api.github.com
    effect: maybe
`)

	_, err := policy.Load(path)
	if err == nil {
		t.Fatal("expected a validation error for an invalid effect")
	}
}

func TestLoad_MissingRequiredField(t *testing.T) {
	path := writeTempPolicyFile(t, `
policies:
  - destination: api.github.com
    effect: allow
`)

	_, err := policy.Load(path)
	if err == nil {
		t.Fatal("expected a validation error for a missing workload field")
	}
}
