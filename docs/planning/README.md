# Planning

## Portainer × account.1crm.io SSO & branding

Umbrella (one-crm workspace): [`../../../.planning/portainer-account-sso/MASTER-PLAN.md`](../../../.planning/portainer-account-sso/MASTER-PLAN.md) · [FEATURE-INDEX](../../../.planning/portainer-account-sso/FEATURE-INDEX.md) · [PROGRESS](../../../.planning/portainer-account-sso/PROGRESS.md)

This repo (`portainer`) owns:

| Feature | Slug | Notes |
|---|---|---|
| account-branding | [`account-branding`](../../../.planning/portainer-account-sso/features/account-branding/SUBPLAN.md) | Login/splash/sidebar tokens from live **account.1crm.io**; living spec remains [`../features/developer-console-branding/`](../features/developer-console-branding/) |
| account-sso (fork) | [`account-sso`](../../../.planning/portainer-account-sso/features/account-sso/SUBPLAN.md) | PKCE + OAuth preset + login CTA; OpenIddict client registration is in **user-service** |

This repo does **not** own NPM/DNS for `develop.1crm.io` (see `docker-composes` planning pointer + umbrella `develop-domain`).
