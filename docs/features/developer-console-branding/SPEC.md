# SPEC — Developer Console branding

## Goal

Present the Portainer CE UI as **1CRM Developer Console**, matching 1CRM Fuse auth layout patterns (split form + dark hero), not stock Portainer marketing chrome.

## Behavior

- Login page: left credentials panel titled “Developer Console” / “Sign in with your 1CRM account”; right hero “1CRM / Developer Console”.
- Sidebar brand: “1CRM” + “Developer Console” (text) when no custom LogoURL is set.
- Footer CE label: “1CRM Developer Console”.
- “Upgrade to Business Edition” button remains removed.
- Upstream version check remains enabled (footer can show “New version available”).

## Non-goals

- Full Fuse Angular rewrite of Portainer.
- Disabling upstream version polling.
