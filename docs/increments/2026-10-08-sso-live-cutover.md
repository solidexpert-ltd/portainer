# Increment 2026-10-08 — SSO live cutover

**Shipped:** Production Developer Console OAuth enabled (`AuthenticationMethod=3`) against `portainer-developer-console` on account/1crm IdP; image `solidexpert/portainer-ce:12` (claim fallbacks + space-split scopes). AutoCreateUsers=true for first ops cutover; UserIdentifier=`preferred_username`.
**Why:** Complete real Portainer↔1CRM SSO so operators can sign in with 1CRM tokens (prior image had PKCE but AuthMethod stayed internal).
**Spec:** `docs/features/account-sso/SPEC.md`
**CI:** `solidexpert-ltd/portainer` run `#12` https://github.com/solidexpert-ltd/portainer/actions/runs/37737927279
**Image:** `solidexpert/portainer-ce:12`
**Prod swap:** `portainer` (backup `portainer-old10`)
**Перезалить:** `portainer` (standalone / `portainer/portainer-ce.stack.yml`)
**Ops:** `.planning/portainer-account-sso/features/account-sso/WORK-HANDOFF-LIVE.md`; runbook `docker-composes/production/portainer/SSO-RUNBOOK.md`
