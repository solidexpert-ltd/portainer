# Increment 2026-10-07 — Login bootstrap fix

**Shipped:** Fix blank login (Angular failed to bootstrap: `ui.bootstrap` / vendor side-effects not loaded). Eager `import './vendors'`, resilient `require.context`, safer without-annoying overlay, auth submit hardening.
**Why:** Production login showed a blank page after the Developer Console image deploy.
**Spec:**
- `docs/features/developer-console-branding/SPEC.md`
**CI:** `solidexpert-ltd/portainer` run `#8` https://github.com/solidexpert-ltd/portainer/actions/runs/37670258001
**Image:** `solidexpert/portainer-ce:8` (also `:2.45.1`, `:latest`)
**Prod swap:** `portainer` (backup `portainer-old5-before8`)
**Перезалить:** `portainer` (standalone container on host)
