# Increment 2026-10-08 — Account branding + SSO (PKCE)

**Shipped:** Developer Console login/sidebar aligned to account.1crm.io (accent `#526ed3`, vendored logos/favicons); native OAuth **PKCE** + **1CRM** provider preset + “Sign in with 1CRM” CTA. Production AuthenticationMethod left on internal login until ops sets IdP secret and applies SSO-RUNBOOK.
**Why:** Match account chrome on `develop.1crm.io` and enable SSO with PKCE required by user-service.
**Spec:**
- `docs/features/developer-console-branding/SPEC.md`
- `docs/features/account-sso/SPEC.md`
**CI:** _(filled after push)_
**Image:** _(filled after CI)_
**Prod swap:** _(filled after swap)_
**Перезалить:** `portainer` (standalone / `portainer/portainer-ce.stack.yml`)
