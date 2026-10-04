package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/audit"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/identity"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
)

// TargetParam is the query param carrying the destination URL to call, e.g.
// GET /proxy?target=https://api.github.com/repos/example/project
const TargetParam = "target"

type Handler struct {
	engine              *policy.Engine
	audit               *audit.Logger
	client              *http.Client
}

func NewHandler(engine *policy.Engine, auditLogger *audit.Logger) *Handler {
	return &Handler{
		engine: engine,
		audit:  auditLogger,
		client: &http.Client{},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := newRequestID()
	w.Header().Set("X-Request-Id", requestID)
	method := r.Method

	workload, err := identity.FromRequest(r)
	if err != nil {
		h.deny(w, requestID, "", "", method, http.StatusUnauthorized, "missing or invalid workload identity: "+err.Error())
		return
	}
	target := r.URL.Query().Get(TargetParam)
	targetURL, err := h.validateTarget(target)
	if err != nil {
		h.deny(w, requestID, workload, hostForAudit(target), method, http.StatusBadRequest, err.Error())
		return
	}
	destination := targetURL.Hostname()

	decision := h.engine.Evaluate(workload, destination, method)
	h.audit.Record(requestID, workload, destination, method, decision.Effect, decision.Reason)
	if decision.Effect == policy.EffectDeny {
		http.Error(w, fmt.Sprintf("denied: %s", decision.Reason), http.StatusForbidden)
		return
	}
	h.forward(w, r, targetURL, requestID)
}

func (h *Handler) forward(w http.ResponseWriter, r *http.Request, target *url.URL, requestID string) {
	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), r.Body)
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}

	resp, err := h.client.Do(outReq)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("X-Request-Id", requestID)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func hostForAudit(target string) string {
	if u, err := url.Parse(target); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return target
}

func (h *Handler) deny(w http.ResponseWriter, requestID, workload, destination, method string, status int, reason string) {
	h.audit.Record(requestID, workload, destination, method, policy.EffectDeny, reason)
	http.Error(w, fmt.Sprintf("denied: %s", reason), status)
}

func newRequestID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func (h *Handler) validateTarget(target string) (*url.URL, error) {
	if target == "" {
		return nil, fmt.Errorf("missing %q query paramter", TargetParam)
	}
	u, err := url.Parse(target)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid target url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported target scheme %q", u.Scheme)
	}
	return u, nil
}
