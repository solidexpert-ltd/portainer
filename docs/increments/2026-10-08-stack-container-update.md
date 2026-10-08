# Increment 2026-10-08 — Single-service stack image update

**Shipped:** `POST /api/stacks/container-image` accepts a container name and an image tag, patches that one compose service, and compose-ups only that service.
**Why:** Git and CI need to roll one service without a Portainer whole-stack update.
**Spec:**
- `docs/features/stack-container-update/SPEC.md` — token check, one-service image patch, single-service compose up
**CI:** `solidexpert-ltd/portainer` run `#21` https://github.com/solidexpert-ltd/portainer/actions/runs/37814422526
**Image:** `solidexpert/portainer-ce:21`
**Prod swap:** `portainer` (backup `portainer-old18`)
**Перезалить:** `portainer` in stack `portainer/portainer-ce.stack.yml`
