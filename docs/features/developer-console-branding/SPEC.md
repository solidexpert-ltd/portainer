# SPEC — Developer Console branding

## Goal

Present the Portainer CE UI as **1CRM Developer Console**, visually aligned with live **account.1crm.io** (light sign-in chrome, Taiga accent CTA, vendored `logo-text` mark) — not stock Portainer marketing chrome and not invented Fuse hex values.

Token source of truth: `.planning/portainer-account-sso/features/account-branding/BRAND-TOKENS.md` (fetched from `https://account.1crm.io/sign-in`).

## Behavior

- Login page: left credentials panel with vendored `logo-text.svg`, title “Developer Console”, subtitle “Sign in with your 1CRM account”; primary CTA `#526ed3` (`--tui-background-accent-1`); fonts Inter + Manrope; right hero slate `#1e293b` with `logo-text-on-dark.svg`.
- OAuth + internal login form structure preserved for later SSO (OAuth buttons, divider, “Use internal authentication”, username/password).
- Sidebar brand: vendored `logo-text-on-dark.svg` + “Developer Console” when no custom LogoURL; collapsed sidebar keeps the mark, hides subtitle.
- Footer CE label: “1CRM Developer Console”.
- Document title / splash / favicons: account favicon family (`favicon-16x16.png`, `favicon-32x32.png`, matching ico/apple-touch); splash shows `logo-text.svg` + “Developer Console”.
- “Upgrade to Business Edition” button remains removed.
- Upstream version check remains enabled (footer can show “New version available”).

## Bootstrap / login reliability

- `app/index.js` must eagerly `import './vendors'` so Angular modules (`ui.bootstrap`, etc.) register before `ng-app` bootstrap.
- `require.context` loading of legacy `*.js` must not abort the whole walk on a single failure.
- Overlay (`build/without-annoying`) must not use broad `button:has(...)` CSS that can hide auth controls; `insertBefore` patch must preserve `this` binding.
- Overlay safety (2026-10-08): injected CSS only targets sidebar Upgrade BE / `.be-indicator*` / `.limited-be`; does **not** match `.dc-btn`, OAuth anchors, or auth submit. No overlay change required for account branding.

## Non-goals

- Full Fuse / Taiga Angular rewrite of Portainer.
- SSO/OIDC wiring (separate feature).
- NPM / domain cutover.
- Disabling upstream version polling.
- Hotlinking `account.1crm.io` assets at runtime.
