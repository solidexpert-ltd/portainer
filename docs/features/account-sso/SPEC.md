# SPEC — account-sso (1CRM Developer Console)

## Behavior

- Portainer can authenticate via native OAuth against **account.1crm.io** (user-service OpenIddict).
- Authorize URL is built without a server-side PKCE challenge; the browser generates S256 `code_challenge` / `code_verifier` and sends `codeVerifier` on `POST /api/auth/oauth/validate`.
- Provider preset **1CRM** (`onecrm`): authorize/token/userinfo/logout on `1crm.io/api/user/connect/*`, scopes `openid user:email user:firstName user:lastName offline_access`, `UserIdentifier=email`, `AuthStyle=InParams`.
- Preset defaults: SSO enabled, **`OAuthAutoCreateUsers=false`** (operators must exist in Portainer).
- Login chrome shows **Sign in with 1CRM** when the configured OAuth provider is 1CRM; internal username/password remains available until ops flips AuthenticationMethod.
- IdP client id: `portainer-developer-console`. Redirect URIs: `https://develop.1crm.io/`, `https://develop.1crm.io`, `https://185.66.69.55:9443/`.

## Ops / production

- Live OAuth cutover requires: this image deployed, IdP client seeded, **hashed** client secret set via seeder env (`Oidc:PortainerDeveloperConsole:ClientSecret`), then settings per `docker-composes/production/portainer/SSO-RUNBOOK.md`.
- Do **not** flip AuthenticationMethod to OAuth without a real client secret.
- Client secret is never stored in git.

## Out of scope

- Auto-provisioning Portainer users from IdP (`OAuthAutoCreateUsers=true`).
- Changing NPM / `develop.1crm.io` edge (see develop-domain).
- Inventing or committing a client secret.
