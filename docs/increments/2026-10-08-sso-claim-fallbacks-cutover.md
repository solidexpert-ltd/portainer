# Increment 2026-10-08 — SSO claim fallbacks + live OAuth cutover

**Shipped:** `GetUsername` falls back through `email` / `preferred_username` / `phone_number` / `sub`; OAuth scopes split on spaces and commas; 1CRM preset uses `preferred_username`. Production Portainer set to `AuthenticationMethod=3` with public IdP client + PKCE.
**Why:** Phone-first 1CRM users often have empty email; Portainer must accept 1CRM tokens and complete real SSO.
**Spec:** `docs/features/account-sso/SPEC.md`
**CI:** `solidexpert-ltd/portainer` run `#12` (claim fix) — image `:12`; prod later on `:13`
**Image:** `solidexpert/portainer-ce:12` / live `:13`
**Prod swap:** `portainer` → `:12`/`:13`; IdP `solidex.users.service` → `user-service.net:262` (SqlServer hotfix for ID2083)
**Перезалить:** `portainer` (stack `portainer-ce.stack.yml`); `solidex.users.service` → `:262` when Hub has the hotfix tag
