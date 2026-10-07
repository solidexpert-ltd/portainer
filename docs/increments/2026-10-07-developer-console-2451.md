# Increment 2026-10-07 — 2.45.1 + Developer Console login

**Shipped:** Portainer CE fork merged to upstream 2.45.1; login/sidebar branded as 1CRM Developer Console; BE upsell removed; upstream version nag kept.
**Why:** Drop Portainer marketing UI, stay on current LTS, keep watching public releases.
**Spec:**
- `docs/features/developer-console-branding/SPEC.md`
**CI:** `solidexpert-ltd/portainer` run `#5` https://github.com/solidexpert-ltd/portainer/actions/runs/37656403677
**Image:** `solidexpert/portainer-ce:5` (also `:2.45.1`, `:latest`)
**Prod swap:** `portainer` (backup `portainer-oldportainer-ce-latest`)
**Перезалить:** `portainer` (standalone container on host)