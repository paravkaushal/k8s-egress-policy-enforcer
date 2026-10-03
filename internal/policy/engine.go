package policy

import (
	"fmt"
	"strings"
)

// Decision is the outcome of evaluating a request
type Decision struct {
	Effect  Effect
	Reason  string
	Matched *Policy
}

type Engine struct {
	policies []Policy
}

func NewEngine(cfg *Config) *Engine {
	return &Engine{
		policies: cfg.Policies,
	}
}

// Evaluate decides whether workload may call destination with method, returning the winning policy and the reason.
func (e *Engine) Evaluate(workload, destination, method string) Decision {
	method = strings.ToUpper(method)
	destination = strings.ToLower(destination)

	var best *Policy
	bestScore := -1
	tie := false

	for i := range e.policies {
		p := &e.policies[i]
		score, ok := matchScore(p, workload, destination, method)
		if !ok {
			continue
		}
		switch {
		case score > bestScore:
			best, bestScore, tie = p, score, false
		case score == bestScore:
			if p.Effect != best.Effect {
				tie = true
				// Deny wins the tie
				if p.Effect == EffectDeny {
					best = p
				}
			}
		}
	}
	if best == nil {
		return Decision{Effect: EffectDeny, Reason: "no matching policy; default deny"}
	}
	reason := fmt.Sprintf("matched policy workload=%q destination=%q effect=%s", best.Workload, best.Destination, best.Effect)
	if tie {
		reason = fmt.Sprintf("conflicting policies at equal specificity; deny takes precendence (workload=%q, destination=%q)", best.Workload, best.Destination)
	}
	return Decision{Effect: best.Effect, Reason: reason, Matched: best}
}

func matchScore(p *Policy, workload, destination, method string) (int, bool) {
	workloadScore, ok := scoreExactOrWildCard(p.Workload, workload)
	if !ok {
		return 0, false
	}
	destScore, ok := scoreExactOrWildCard(p.Destination, destination)
	if !ok {
		return 0, false
	}
	methodScore, ok := scoreMethods(p.Methods, method)
	if !ok {
		return 0, false
	}
	return workloadScore*100 + destScore*10 + methodScore, true
}

func scoreExactOrWildCard(pattern, value string) (int, bool) {
	switch {
	case pattern == "*":
		return 1, true
	case strings.EqualFold(pattern, value):
		return 2, true
	default:
		return 0, false
	}
}

func scoreMethods(methods []string, method string) (int, bool) {
	if len(methods) == 0 {
		return 0, true
	}
	for _, m := range methods {
		if strings.EqualFold(m, method) {
			return 1, true
		}
	}
	return 0, false
}
