# Design Decisions, Assumptions & Failure Behavior

## Design Decisions

- **Explicit endpoint, not a transparent proxy.** The client calls `/proxy?target=...`
  directly rather than the enforcer silently intercepting raw egress traffic. This is
  testable with plain `curl`/`httptest` and keeps the implementation small; please see
  [Kubernetes/Envoy/Istio Integration](PRODUCTION.md#kubernetesenvoyistio-integration) for how
  a real deployment removes the app's ability to opt out of this path.
- **Policy file loaded once at startup**, not watched for changes. A restart is required to
  pick up edits - explicitly acceptable per the assignment; please see
  [Policy Updates](PRODUCTION.md#policy-updates-in-production) for the production alternative.
- **Score-all, highest-specificity-wins matching** (please see [Policy Evaluation Engine](POLICY.md))
  instead of first-match-wins, so policy order in the file never changes the outcome.
- **Deny wins ties, no match defaults to deny** - the engine always fails closed.
## Assumptions

- **Workload identity**: the enforcer trusts an `X-Workload-Id` header set by the caller. This
  is the simplest mechanism that satisfies the requirement, but it is **not spoof-proof** by
  itself - anything in front of the request path must prevent the workload from setting this
  header itself (please see [Kubernetes/Envoy/Istio Integration](PRODUCTION.md) for the production
  answer: mTLS-derived identity).
- **Policy file is locally trusted input**: it's read from disk at startup, not accepted over
  the network, so no additional authn/authz is applied to it in this assignment.
- **One policy file, one process**: no multi-tenancy or per-namespace policy partitioning is
  implemented; all workloads share one flat policy list.
## Failure Behavior

| Failure                                          | Behavior                                                        |
|---------------------------------------------------|------------------------------------------------------------------|
| Policy file missing / invalid YAML / invalid policy | Process logs the error and **exits immediately** - fails closed, never serves traffic with no or broken policy data. |
| Missing/invalid `X-Workload-Id` header            | `401`, audited as `deny`.                                        |
| Missing/invalid `target`, unsupported scheme | `400`, audited as `deny`.                        |
| No policy matches                                 | `403`, audited as `deny`, reason `"no matching policy; default deny"`. |
| Policy matches, effect `deny`                     | `403`, audited as `deny` with the matched policy in the reason.  |
| Policy matches, effect `allow`, but upstream is unreachable | `502`; the **policy decision was still `allow`** and is audited as such - the audit trail reflects the policy outcome, not network availability. |
| Process receives `SIGINT`/`SIGTERM`               | Stops accepting new connections and drains in-flight requests for up to 5s before exiting. |
