# Increment 2026-10-17 — Docker image-pull timeout

**Shipped:** Agent image pulls on `POST /api/stacks/container-image` wait up to 10 minutes instead of 60 seconds.
**Why:** Crawler pulls through the VM 113 agent were cut off at `Client.Timeout` (`defaultDockerRequestTimeout` was 60s) and returned **422** `Failed to pull image`.
**Spec:**
- `docs/features/stack-container-update/SPEC.md` — step 7 uses `DefaultDockerRequestTimeout` (10 minutes) for the pull stream
**CI:** `solidexpert-ltd/portainer` run `#23` https://github.com/solidexpert-ltd/portainer/actions/runs/39030086476
**Image:** `solidexpert/portainer-ce:23`
**Prod swap:** `portainer` (backup `portainer-old21`)
**Перезалить:** `portainer` in stack `portainer/portainer-ce.stack.yml`
