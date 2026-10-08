# 2026-10-08 — SSO on account.1crm.io + classic admin login

**Shipped:** Developer Console OAuth endpoints point at `account.1crm.io` (authorize/token/userinfo/logout). OAuth client secret can be cleared with `-` (public IdP + PKCE). Break-glass admin password reset and verified via `/api/auth`. IdP client `portainer-developer-console` is **public**. Portainer image `:18`; user-service IdP hotfix `:262` (ID2083).

**Ops:** admin password only on host `/home/docker/secrets/portainer-sso/breakglass-admin.txt` — change after first login.
