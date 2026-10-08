# Stack container update

**Purpose:** Git or CI sets one compose service's image tag and recreates only that service.
**Actors:** Git/CI. A Portainer JWT is not used. An agent is not in the request path.

## Code

- `api/http/handler/stacks/stack_container_image.go` — token, lookup, pull, file write, compose up
- `api/http/handler/stacks/handler.go` — `POST /stacks/container-image` registered before `/stacks/{id}`
- `api/stacks/stackutils/image_tag.go` — line edit of one `image:` scalar
- `pkg/libstack/compose/composeplugin.go` — one-service up drops dependencies and does not recreate siblings

## API

- `POST /api/stacks/container-image` — header `X-Stack-Update-Token` or JSON `token`; body `container` + `version` (tag only)

## Spec

Current behavior: [SPEC.md](SPEC.md)
