# Increment 2026-10-07 — Login bootstrap fix

**Shipped:** Fix blank login (Angular failed to bootstrap: `ui.bootstrap` / vendor side-effects not loaded). Eager `import './vendors'`, resilient `require.context`, safer without-annoying overlay, auth submit hardening.
**Why:** Production login showed a blank page after the Developer Console image deploy.
**Spec:**
- `docs/features/developer-console-branding/SPEC.md`
**CI:** (filled after green run)
**Image:** (filled after publish)
**Prod swap:** `portainer`
**Перезалить:** `portainer` (standalone container on host)
