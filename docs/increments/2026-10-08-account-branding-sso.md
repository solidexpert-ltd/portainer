# Increment 2026-10-08 — Account branding + SSO (PKCE)

**Shipped:** Developer Console login/sidebar aligned to account.1crm.io (accent `#526ed3`, vendored logos/favicons); native OAuth **PKCE** + **1CRM** provider preset + “Sign in with 1CRM” CTA. Production AuthenticationMethod left on internal login until ops sets IdP secret and applies SSO-RUNBOOK.
**Why:** Match account chrome on `develop.1crm.io` and enable SSO with PKCE required by user-service.
**Spec:**
- `docs/features/developer-console-branding/SPEC.md`
- `docs/features/account-sso/SPEC.md`
**CI:** `solidexpert-ltd/portainer` run `#10` https://github.com/solidexpert-ltd/portainer/actions/runs/37708174645
**Image:** `solidexpert/portainer-ce:10` (also `:2.45.1`, `:latest`)
**Prod swap:** `portainer` (backup `portainer-old8`)
**Перезалить:** `portainer` (standalone / `portainer/portainer-ce.stack.yml`)
