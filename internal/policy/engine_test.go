package policy_test

import (
	"testing"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
)

func samplePolicies() []policy.Policy {
	return []policy.Policy{
		{Workload: "payment-service", Destination: "api.github.com", Methods: []string{"GET"}, Effect: policy.EffectAllow},
		{Workload: "payment-service", Destination: "api.slack.com", Methods: []string{"POST"}, Effect: policy.EffectAllow},
		{Workload: "payment-service", Destination: "*", Effect: policy.EffectDeny},
	}
}

func TestEvaluate_AllowedRequest(t *testing.T) {
	engine := policy.NewEngine(&policy.Config{Policies: samplePolicies()})
	decision := engine.Evaluate("payment-service", "api.github.com", "GET")
	if decision.Effect != policy.EffectAllow {
		t.Fatalf("expected allow, got %s (reason: %s)", decision.Effect, decision.Reason)
	}
}

func TestEvaluate_DeniedRequest_WrongMethodFallsToWildcard(t *testing.T) {
	engine := policy.NewEngine(&policy.Config{Policies: samplePolicies()})
	decision := engine.Evaluate("payment-service", "api.slack.com", "GET")
	if decision.Effect != policy.EffectDeny {
		t.Fatalf("expected deny, got %s", decision.Effect)
	}
	if decision.Matched == nil || decision.Matched.Destination != "*" {
		t.Fatalf("expected the wildcard catch-all to match, got %+v", decision.Matched)
	}
}

func TestEvaluate_UnknownWorkload(t *testing.T) {
	engine := policy.NewEngine(&policy.Config{Policies: samplePolicies()})
	decision := engine.Evaluate("some-other-service", "api.github.com", "GET")
	if decision.Effect != policy.EffectDeny {
		t.Fatalf("expected deny for unknown workload, got %s", decision.Effect)
	}
	if decision.Matched != nil {
		t.Fatalf("expected default deny (no matched policy), got %+v", decision.Matched)
	}
}

func TestEvaluate_UnknownDestination_NoWildcard(t *testing.T) {
	policies := []policy.Policy{
		{Workload: "payment-service", Destination: "api.github.com", Methods: []string{"GET"}, Effect: policy.EffectAllow},
	}
	engine := policy.NewEngine(&policy.Config{Policies: policies})
	decision := engine.Evaluate("payment-service", "unknown.example.com", "GET")
	if decision.Effect != policy.EffectDeny {
		t.Fatalf("expected deny for unknown destination, got %s", decision.Effect)
	}
	if decision.Matched != nil {
		t.Fatalf("expected default deny (no matched policy), got %+v", decision.Matched)
	}
}

func TestEvaluate_UnknownDestination_WildcardCatchAll(t *testing.T) {
	engine := policy.NewEngine(&policy.Config{Policies: samplePolicies()})
	decision := engine.Evaluate("payment-service", "unknown.example.com", "GET")
	if decision.Effect != policy.EffectDeny {
		t.Fatalf("expected deny via wildcard catch-all, got %s", decision.Effect)
	}
	if decision.Matched == nil || decision.Matched.Destination != "*" {
		t.Fatalf("expected match against the wildcard policy, got %+v", decision.Matched)
	}
}

func TestEvaluate_ConflictingPolicies_DenyWinsTie(t *testing.T) {
	policies := []policy.Policy{
		{Workload: "payment-service", Destination: "api.github.com", Effect: policy.EffectAllow},
		{Workload: "payment-service", Destination: "api.github.com", Effect: policy.EffectDeny},
	}
	engine := policy.NewEngine(&policy.Config{Policies: policies})
	decision := engine.Evaluate("payment-service", "api.github.com", "GET")
	if decision.Effect != policy.EffectDeny {
		t.Fatalf("expected deny to win an equally specific conflict, got %s", decision.Effect)
	}
}

func TestEvaluate_ExactWorkloadBeatsWildcardWorkload(t *testing.T) {
	policies := []policy.Policy{
		{Workload: "*", Destination: "api.github.com", Effect: policy.EffectDeny},
		{Workload: "payment-service", Destination: "api.github.com", Effect: policy.EffectAllow},
	}
	engine := policy.NewEngine(&policy.Config{Policies: policies})
	decision := engine.Evaluate("payment-service", "api.github.com", "GET")
	if decision.Effect != policy.EffectAllow {
		t.Fatalf("expected exact workload match to outrank the wildcard, got %s (%s)", decision.Effect, decision.Reason)
	}
}

func TestEvaluate_CaseInsensitiveDestinationAndMethod(t *testing.T) {
	engine := policy.NewEngine(&policy.Config{Policies: samplePolicies()})
	decision := engine.Evaluate("payment-service", "API.GitHub.com", "get")
	if decision.Effect != policy.EffectAllow {
		t.Fatalf("expected case-insensitive match to allow, got %s (%s)", decision.Effect, decision.Reason)
	}
}

func TestEvaluate_EmptyMethodsListMatchesAnyMethod(t *testing.T) {
	policies := []policy.Policy{
		{Workload: "payment-service", Destination: "api.github.com", Effect: policy.EffectAllow},
	}
	engine := policy.NewEngine(&policy.Config{Policies: policies})
	for _, method := range []string{"GET", "POST", "DELETE"} {
		decision := engine.Evaluate("payment-service", "api.github.com", method)
		if decision.Effect != policy.EffectAllow {
			t.Fatalf("method %s: expected allow with empty Methods list, got %s", method, decision.Effect)
		}
	}
}

func TestEvaluate_NoPoliciesLoaded(t *testing.T) {
	engine := policy.NewEngine(&policy.Config{})
	decision := engine.Evaluate("payment-service", "api.github.com", "GET")
	if decision.Effect != policy.EffectDeny {
		t.Fatalf("expected default deny with no policies loaded, got %s", decision.Effect)
	}
	if decision.Matched != nil {
		t.Fatalf("expected no matched policy, got %+v", decision.Matched)
	}
}
