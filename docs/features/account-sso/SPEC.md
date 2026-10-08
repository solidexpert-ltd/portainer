# SPEC — account-sso (1CRM Developer Console)

## Behavior

- Portainer can authenticate via native OAuth against **account.1crm.io** (user-service OpenIddict).
- Authorize URL is built without a server-side PKCE challenge; the browser generates S256 `code_challenge` / `code_verifier` and sends `codeVerifier` on `POST /api/auth/oauth/validate`.
- Provider preset **1CRM** (`onecrm`): authorize/token/userinfo/logout on `1crm.io/api/user/connect/*`, scopes `openid user:email user:firstName user:lastName offline_access`, `UserIdentifier=preferred_username`, `AuthStyle=InParams`.
- Username extraction tries configured claim, then `email` → `preferred_username` → `phone_number` → `sub` (phone-first IdP users often have empty email).
- OAuth scopes may be space- or comma-separated; Portainer splits both.
- Preset defaults: SSO enabled; first cutover uses **`OAuthAutoCreateUsers=true`** (see runbook).
- Login chrome shows **Sign in with 1CRM** when the configured OAuth provider is 1CRM; internal username/password remains available until ops flips AuthenticationMethod.
- IdP client id: `portainer-developer-console` (**public** + PKCE on prod). Redirect URIs: `https://develop.1crm.io/`, `https://develop.1crm.io`, `https://185.66.69.55:9443/`.

## Ops / production

- Live OAuth cutover: image with PKCE + claim fallbacks, IdP client seeded as public+PKCE, settings per `docker-composes/production/portainer/SSO-RUNBOOK.md`.
- Portainer ClientSecret must be empty while IdP client is public.
- Live handoff: `.planning/portainer-account-sso/features/account-sso/WORK-HANDOFF-LIVE.md`.

## Out of scope

- Changing NPM / `develop.1crm.io` edge (see develop-domain).
- Inventing or committing client secrets.
