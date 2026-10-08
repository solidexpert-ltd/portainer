# Feature: account-sso

Portainer (**1CRM Developer Console**) signs in through **account.1crm.io** using native OAuth + PKCE against user-service OpenIddict.

## Status

Implementation in the Portainer fork (PKCE, 1CRM provider preset, login CTA). Production OAuth enablement is ops-only — see the runbook.

## Ops runbook

Canonical copy: [`docker-composes/production/portainer/SSO-RUNBOOK.md`](../../../../docker-composes/production/portainer/SSO-RUNBOOK.md).

## Client (IdP)

| Field | Value |
| --- | --- |
| ClientId | `portainer-developer-console` |
| Redirect URIs | `https://develop.1crm.io/`, `https://develop.1crm.io`, `https://185.66.69.55:9443/` |
| PKCE | Required (`RequireProofKeyForCodeExchange` on user-service) |

Secret: ops vault only. Seed: user-service `Migrations/Scripts/20261008030000_SeedPortainerDeveloperConsole.sql`.

## Image

Rebuild/push `solidexpert/portainer-ce` (or GHCR equivalent) after merging PKCE + auth UI changes before flipping AuthenticationMethod to OAuth on production.

## Planning

`.planning/portainer-account-sso/features/account-sso/` — SUBPLAN, IMPLEMENTATION, WORK-HANDOFF.
