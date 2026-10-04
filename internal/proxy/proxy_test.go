package proxy_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/audit"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/identity"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/policy"
	"github.com/paravkaushal/k8s-egress-policy-enforcer/internal/proxy"
)

func newTestHandler(policies []policy.Policy, auditBuf *bytes.Buffer) *proxy.Handler {
	engine := policy.NewEngine(&policy.Config{Policies: policies})
	h := proxy.NewHandler(engine, audit.New(auditBuf))
	return h
}

func hostOf(t *testing.T, rawURL string) string {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}
	return u.Hostname()
}

func lastAuditRecord(t *testing.T, auditBuf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(auditBuf.Bytes()), []byte("\n"))
	if len(lines) == 0 || len(lines[0]) == 0 {
		t.Fatal("expected at least one audit record, got none")
	}
	var record map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &record); err != nil {
		t.Fatalf("parsing audit record: %v", err)
	}
	for _, field := range []string{"timestamp", "request_id", "workload", "destination", "method", "decision", "reason"} {
		if _, ok := record[field]; !ok {
			t.Fatalf("audit record missing field %q: %v", field, record)
		}
	}
	return record
}

func TestServeHTTP_AllowedRequestIsForwarded(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response"))
	}))
	defer upstream.Close()

	var auditBuf bytes.Buffer
	h := newTestHandler([]policy.Policy{
		{Workload: "payment-service", Destination: hostOf(t, upstream.URL), Methods: []string{"GET"}, Effect: policy.EffectAllow},
	}, &auditBuf)

	req := httptest.NewRequest(http.MethodGet, "/proxy?target="+upstream.URL, nil)
	req.Header.Set(identity.Header, "payment-service")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != "upstream response" {
		t.Fatalf("unexpected forwarded body: %q", body)
	}
	if record := lastAuditRecord(t, &auditBuf); record["decision"] != "allow" {
		t.Fatalf("expected audit decision allow, got %v", record["decision"])
	}
}

func TestServeHTTP_DeniedRequestReturns403(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("upstream should not be called for a denied request")
	}))
	defer upstream.Close()

	var auditBuf bytes.Buffer
	h := newTestHandler([]policy.Policy{
		{Workload: "payment-service", Destination: "*", Effect: policy.EffectDeny},
	}, &auditBuf)

	req := httptest.NewRequest(http.MethodGet, "/proxy?target="+upstream.URL, nil)
	req.Header.Set(identity.Header, "payment-service")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if record := lastAuditRecord(t, &auditBuf); record["decision"] != "deny" {
		t.Fatalf("expected audit decision deny, got %v", record["decision"])
	}
}

func TestServeHTTP_MissingIdentityReturns401(t *testing.T) {
	var auditBuf bytes.Buffer
	h := newTestHandler(nil, &auditBuf)

	req := httptest.NewRequest(http.MethodGet, "/proxy?target=https://api.github.com/x", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServeHTTP_MissingTargetReturns400(t *testing.T) {
	var auditBuf bytes.Buffer
	h := newTestHandler(nil, &auditBuf)

	req := httptest.NewRequest(http.MethodGet, "/proxy", nil)
	req.Header.Set(identity.Header, "payment-service")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
