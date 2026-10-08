# SPEC — Container CPU, memory, and block I/O columns

## Goal

Show optional per-container CPU, memory, and block I/O on the Docker containers table and on the stack details containers table, without taking down the rest of the page.

## Behavior

- Columns `cpu`, `memory`, and `blockIO` are hidden by default.
- Saved table settings from before these columns existed must not leave them visible on the next visit. Each table (`containers`, `stack-containers`) hides them once; a later choice in the column menu is kept.
- Metric cells read `isMetricsEnabled` from the row context. The query runs only when at least one of those columns is shown.
- The metrics query uses `withError` from `@/react-tools/react-query` (the helper that actually exists). A missing export must not throw while rendering a cell.
- A non-array metrics payload or a non-finite CPU percent renders as empty / "Not Available". It must not unmount the stack details view.
- Data comes from `GET /endpoints/{id}/metrics/containers/current`.

## Non-goals

- Changing how Docker stats are calculated.
- Showing the columns by default.
