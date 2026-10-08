# Increment 2026-10-08 — Stack details no longer crash on container metrics

**Shipped:** Stack details (and the containers table) render again when saved column settings make the CPU / memory / block I/O columns visible. The metrics query calls `withError` instead of the missing `withGlobalError`. Those columns are hidden once for existing saved table settings.
**Why:** Opening a stack mounted metric cells, `withGlobalError` threw during render, and the stack view unmounted to a blank page.
**Spec:** `docs/features/container-resource-columns/SPEC.md`
**CI:** `solidexpert-ltd/portainer` run `#13` https://github.com/solidexpert-ltd/portainer/actions/runs/37739735977
**Image:** `solidexpert/portainer-ce:13` (`2.45.1-8b577d2`)
**Prod swap:** `portainer` (backup `portainer-old12`)
**Перезалить:** `portainer` (`portainer/portainer-ce.stack.yml`)
