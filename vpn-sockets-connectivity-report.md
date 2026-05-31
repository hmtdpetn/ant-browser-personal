# VPN Sockets Connectivity Report

## 1. Scope

This report targets the proxy connectivity path used by the project, especially the `SOCKS/SOCKS5` path that users may describe as "VPN Sockets".

The goal is to answer three questions:

- What problems exist today
- Why they happen
- How to normalize the flow and produce actionable diagnostics instead of blind troubleshooting

## 2. Summary

The current issue is not that SOCKS is completely unsupported. The real problem is that input, storage, validation, testing, and browser runtime use different rules.

This causes several bad outcomes:

- validation passes, but the browser cannot actually use the proxy
- connectivity tests pass, but instance startup still fails
- some alias formats work in one entry path but fail in another
- error messages are too coarse to identify the failed stage

The highest-probability root cause is the handling of authenticated `SOCKS5` as a direct browser proxy.

## 3. Confirmed Problems

### 3.1 Authenticated SOCKS5 is treated as browser-direct

The UI allows entering `socks5://user:pass@host:port` and says it "takes effect directly" without a bridge.

Relevant code:

- [frontend/src/modules/browser/pages/proxyPool/ProxyPoolModals.tsx](d:\code\open_source\ant-chrome\frontend\src\modules\browser\pages\proxyPool\ProxyPoolModals.tsx)
- [frontend/src/modules/browser/pages/proxyPool/helpers.ts](d:\code\open_source\ant-chrome\frontend\src\modules\browser\pages\proxyPool\helpers.ts)
- [backend/internal/proxy/xray.go](d:\code\open_source\ant-chrome\backend\internal\proxy\xray.go)
- [backend/app_instance_start_prepare.go](d:\code\open_source\ant-chrome\backend\app_instance_start_prepare.go)

At startup, the app eventually passes the proxy string into Chromium through `--proxy-server=%s`.

This is problematic because Chromium manual proxy mode is not a reliable path for authenticated proxies, especially authenticated `SOCKSv5`.

Official reference:

- Chromium proxy documentation: <https://chromium.googlesource.com/chromium/src/+/master/net/docs/proxy.md>

Observed impact:

- test may pass
- startup may pass
- browser network access may still fail after launch

### 3.2 Alias handling is inconsistent

The frontend only normalizes `socket://` and `socks://` to `socks5://` in one direct-import path.

Relevant code:

- [frontend/src/modules/browser/pages/proxyPool/helpers.ts](d:\code\open_source\ant-chrome\frontend\src\modules\browser\pages\proxyPool\helpers.ts)

But save logic and backend parsing do not apply the same normalization globally.

Relevant code:

- [backend/app_proxy_save.go](d:\code\open_source\ant-chrome\backend\app_proxy_save.go)
- [backend/internal/proxy/parser.go](d:\code\open_source\ant-chrome\backend\internal\proxy\parser.go)

Observed impact:

- a newly added proxy may be silently fixed
- an edited proxy may not be fixed
- imported or historical data may fail validation or runtime

### 3.3 Validation is weaker than runtime reality

Current validation mainly checks whether backend code can parse the proxy config. It does not check whether Chromium can actually use that config.

Relevant code:

- [frontend/src/modules/browser/pages/BrowserListPage.tsx](d:\code\open_source\ant-chrome\frontend\src\modules\browser\pages\BrowserListPage.tsx)
- [backend/app_instance_start_proxy.go](d:\code\open_source\ant-chrome\backend\app_instance_start_proxy.go)
- [backend/internal/proxy/xray.go](d:\code\open_source\ant-chrome\backend\internal\proxy\xray.go)

Observed impact:

- "supported" does not mean "browser-usable"
- users only discover the failure after launch

### 3.4 Test stack and runtime stack are different

Connectivity tests and IP health checks use Go proxy clients and Mihomo-related logic, while actual browsing uses Chromium.

Relevant code:

- [backend/internal/proxy/http_client.go](d:\code\open_source\ant-chrome\backend\internal\proxy\http_client.go)
- [backend/internal/proxy/speedtest.go](d:\code\open_source\ant-chrome\backend\internal\proxy\speedtest.go)

Observed impact:

- Go test stack can successfully authenticate against SOCKS5
- Chromium runtime can still fail
- users see false positives from proxy tests

### 3.5 Basic TCP connectivity parsing is fragile for authenticated URLs

The base endpoint extraction for connectivity testing does not strip `user:pass@` before using the address.

Relevant code:

- [backend/internal/proxy/utils_parse.go](d:\code\open_source\ant-chrome\backend\internal\proxy\utils_parse.go)

Observed impact:

- authenticated proxy endpoints can be mis-parsed
- the app may incorrectly report low-level connectivity failure

### 3.6 Error reporting is too coarse

The app currently surfaces broad messages such as proxy unavailable or bridge failed, but not the actual failed stage.

Relevant code:

- [frontend/src/App.tsx](d:\code\open_source\ant-chrome\frontend\src\App.tsx)

Observed impact:

- operators cannot tell whether the failure is caused by format, alias, auth, reachability, bridge process, or Chromium compatibility
- troubleshooting becomes trial and error

## 4. Root Cause Analysis

The problems come from a lack of one normalized pipeline.

Today, the system effectively has separate rule sets for:

- input formatting
- persistence
- validation
- speed testing
- browser launch

Because these rule sets do not fully match, the system can produce contradictory outcomes.

The most important contradiction is:

- the test stack supports authenticated SOCKS5
- the browser runtime path does not reliably support the same config

## 5. Normalization Proposal

### 5.1 Canonical proxy formats

Internally, only the following normalized formats should be allowed:

- `direct://`
- `http://host:port`
- `https://host:port`
- `socks5://host:port`
- `vmess://...`
- `vless://...`
- `trojan://...`
- `ss://...`
- Clash YAML
- sing-box-supported protocols already handled by current code

### 5.2 Alias policy

Apply the same normalization in all paths:

- `socks://` -> `socks5://`
- `socket://` -> either reject explicitly or convert globally to `socks5://`

Do not normalize only in the frontend import flow.

Normalization should happen in:

- import
- edit
- save
- load
- pre-launch validation

### 5.3 Runtime mode classification

Every proxy should be classified into exactly one runtime mode:

- `browser_direct`
- `bridge_xray`
- `bridge_singbox`

This classification should be computed once and reused by testing and launch code.

### 5.4 Authentication policy

Recommended policy:

- unauthenticated `http/https/socks5` may use `browser_direct`
- authenticated `SOCKS5` must not use `browser_direct`
- authenticated `HTTP/HTTPS` should be treated as unsupported for direct browser runtime unless explicit browser-level handling is implemented

If the system keeps accepting authenticated direct proxies, it must warn that a successful test does not guarantee browser runtime success.

### 5.5 Dual storage

Each proxy record should preserve:

- `rawProxyConfig`
- `normalizedProxyConfig`

All diagnostics, tests, and launches should use `normalizedProxyConfig`.

## 6. Standard Connectivity Diagnosis Flow

To avoid blind troubleshooting, every test and every launch should go through the same diagnostic stages.

### Stage 1: Normalize

Output:

- raw config
- normalized config
- detected scheme
- whether alias normalization happened
- whether auth is present

### Stage 2: Parse and Validate

Output:

- parse success or failure
- missing fields
- unsupported scheme
- invalid URI or YAML

### Stage 3: Browser Compatibility Check

Output:

- whether Chromium direct runtime can use this proxy
- whether bridge is required

Typical failure example:

- authenticated `socks5://user:pass@host:port`
- parse success
- browser direct incompatible
- resolution: bridge required

### Stage 4: Runtime Mode Decision

Output:

- chosen mode: `browser_direct`, `bridge_xray`, or `bridge_singbox`

### Stage 5: Endpoint Reachability

Output:

- TCP reachability to upstream endpoint
- latency

### Stage 6: Authentication Check

Output:

- whether username/password authentication succeeds
- whether failure is auth-related instead of network-related

### Stage 7: Bridge Check

If bridge mode is required, report:

- bridge binary missing
- config generation failure
- local port conflict
- process start failure
- bridge early exit

### Stage 8: Browser Runtime Verification

The system should explicitly distinguish:

- test stack result
- actual browser runtime result

This is critical because the current false-positive pattern comes from those two stacks being different.

## 7. Recommended Structured Report

Each connectivity test or launch attempt should emit a structured report with at least these fields:

```json
{
  "reportId": "proxy-check-20260413-001",
  "time": "2026-04-13T12:00:00Z",
  "profileId": "profile-001",
  "proxyId": "proxy-001",
  "rawProxyConfig": "socket://user:***@hk.example.com:1080",
  "normalizedProxyConfig": "socks5://user:***@hk.example.com:1080",
  "protocol": "socks5",
  "authMode": "username_password",
  "connectionMode": "browser_direct",
  "testStack": "go+mihomo",
  "runtimeStack": "chromium",
  "lastSuccessStage": "normalize",
  "failedStage": "browser_compatibility",
  "errorCode": "P103_BROWSER_DIRECT_AUTH_UNSUPPORTED",
  "errorMessage": "Authenticated SOCKS5 is not supported in browser direct mode",
  "rootCause": "Authenticated SOCKS5 was treated as a browser-direct proxy",
  "suggestion": "Use a local bridge and let the browser connect to the local no-auth proxy port"
}
```

## 8. Suggested Error Codes

- `P001_ALIAS_NORMALIZED`
- `P101_UNSUPPORTED_SCHEME`
- `P102_INVALID_PROXY_FORMAT`
- `P103_BROWSER_DIRECT_AUTH_UNSUPPORTED`
- `P201_ENDPOINT_UNREACHABLE`
- `P202_PROXY_AUTH_FAILED`
- `P301_BRIDGE_BINARY_MISSING`
- `P302_BRIDGE_PORT_CONFLICT`
- `P303_BRIDGE_START_FAILED`
- `P304_BRIDGE_EXITED`
- `P401_BROWSER_PROXY_APPLY_FAILED`
- `P402_TEST_STACK_MISMATCH`

## 9. Recommended Fix Priority

### P0

- Normalize all proxy schemes in one shared backend path
- Reject or force-bridge authenticated `SOCKS5`
- Add structured connectivity reports for test and launch

### P1

- Fix endpoint parsing for authenticated URLs in low-level connectivity testing
- Mark test results with the actual test stack used
- Add explicit browser-compatibility validation before launch

### P2

- Migrate historical proxy data to normalized formats
- Add a diagnostic detail panel in the UI instead of only top-level error toasts

## 10. Final Judgment

Yes, this flow should be normalized.

If the project keeps the current behavior, users will continue to see:

- proxy test success but browser failure
- inconsistent behavior between import and edit
- low-quality error messages
- repeated blind troubleshooting

The most important engineering change is not "more testing". It is:

- one normalization pipeline
- one runtime mode decision
- one structured diagnostic report

Once those are in place, the team can quickly tell whether the problem is:

- proxy format
- alias issue
- auth issue
- reachability issue
- bridge issue
- Chromium compatibility issue

## 11. Note

This document is based on the current repository code and a review of the Chromium proxy documentation. No code changes are included in this report.
