# Example Requests

The enforcer exposes a single endpoint, `/proxy`, which takes the real destination as a
`target` query parameter and the calling workload as an `X-Workload-Id` header. The HTTP
method used to call `/proxy` is the method forwarded to the destination.

**Allowed** - `payment-service` calling `GET api.github.com`, matching the first policy:

```sh
curl -i -H "X-Workload-Id: payment-service" \
  "http://localhost:8080/proxy?target=https://api.github.com/repos/example/project"
# HTTP/1.1 200 OK  (GitHub's real response is forwarded back as-is)
```

**Denied** - same workload, same destination, but with the wrong method (only `POST` is
allowed to `api.slack.com`), so it falls through to the wildcard deny-all:

```sh
curl -i -H "X-Workload-Id: payment-service" \
  "http://localhost:8080/proxy?target=https://api.slack.com/api/test"
# HTTP/1.1 403 Forbidden
# denied: matched policy workload="payment-service" destination="*" effect=deny
```

**Denied** - unknown workload, no policy matches at all (default deny):

```sh
curl -i -H "X-Workload-Id: unknown-service" \
  "http://localhost:8080/proxy?target=https://api.github.com/repos/example/project"
# HTTP/1.1 403 Forbidden
# denied: no matching policy; default deny
```

**Denied** - missing identity header:

```sh
curl -i "http://localhost:8080/proxy?target=https://api.github.com/x"
# HTTP/1.1 401 Unauthorized
# denied: missing or invalid workload identity: missing workload identity
```

Every attempt above produces one audit line on stdout, e.g.:

```json
{"timestamp":"2026-10-03T11:52:44.417372988+05:30","level":"INFO","msg":"egress_decision","request_id":"c7922bd5a56ee0f8149776f36b5a5dde","workload":"payment-service","destination":"api.slack.com","method":"POST","decision":"allow","reason":"matched policy workload=\"payment-service\" destination=\"api.slack.com\" effect=allow"}
```
