# Stack container update

Git or CI calls Portainer. The Portainer process patches the compose file it stores for that stack and compose-ups that one service. No agent handles the request. This route does not write `docker-composes/production/yammy*.yaml` or any other git checkout.

Do not send this request at the `portainer` container.

## Request

```http
POST /api/stacks/container-image
X-Stack-Update-Token: <secret>
Content-Type: application/json

{"container":"yammy-site","version":"35"}
```

The stacks router is mounted with `StripPrefix("/api")`. The handler path is `POST /stacks/container-image`, registered as a literal so `/stacks/{id}` does not capture it. The bouncer wrapper is `PublicAccess` only so the caller does not need a Portainer JWT. The handler still requires the stack-update token.

`container` is either the compose service key (`yammy-site`) or the Docker name (`yammy-yammy-site-1` or `/yammy-yammy-site-1`). `version` is the tag only (`35`), not `repo:tag`.

The same call with the token in JSON and no header:

```json
{"token":"<secret>","container":"yammy-yammy-site-1","version":"35"}
```

## Auth

Env `PORTAINER_STACK_UPDATE_TOKEN` on the `portainer` process. Unset or empty → **503** `Stack update token is not configured` before the body is trusted and before any lookup or file write.

The compare is `subtle.ConstantTimeCompare`.

- A non-empty `X-Stack-Update-Token` is the only token. A wrong header is **401** even when the JSON `token` is correct. The body token is not a fallback.
- With no header (or an empty header), the JSON `token` is used. Missing or wrong → **401** `Invalid stack update token` before any Docker, stack, or file call.
- Invalid JSON with no matching header is **401**, not 400. **400** `Invalid request payload` is only after the token matches (`container` required, `version` required, or a body that does not decode once the header is already valid).

The token is not written to logs or error bodies.

## Success

**200** body:

```json
{"status":"updated","endpointId":1,"stackId":1,"stackName":"yammy","service":"yammy-site","container":"yammy-yammy-site-1","image":"solidexpert/service.yammy-site:35","previousTag":"34"}
```

| Field | Meaning |
|---|---|
| `status` | `updated` or `unchanged` |
| `endpointId` | Portainer environment that owns the container |
| `stackId` | Stack record id |
| `stackName` | Stack name |
| `service` | Compose key from `com.docker.compose.service` |
| `container` | Docker name with one leading `/` removed |
| `image` | `repository:version`. Repository comes from the stack file, not from the request |
| `previousTag` | Tag that was in the stack file before this call |

`unchanged` means the stack-file tag and the running image already equal `repository:version`. No pull, no file write, no compose up. A digest suffix on the running image (`repo:tag@sha256:…`) still counts as that tag.

If the stack file is already on `version` and the running container is behind, the file is left as-is, the image is pulled, and only that service is composed up. The response is `updated`.

## What the call does

1. Token check.
2. List containers on every Docker Portainer environment (`ContainerList` with all containers, including stopped). Kubernetes environments are skipped. The client is that environment's Docker API, not the Portainer host socket alone. If any environment fails to list, the call is **500** `Unable to list containers on every environment` and does not write. A partial match is not enough.
3. Match the query to Docker `Names` (one leading `/` stripped), `com.docker.compose.service`, or `com.docker.compose.project`. Zero matches → **404**. More than one, including two services in the same project when the query is the project name → **409**. Pass the service key or the Docker name.
4. Load the stack whose name equals `com.docker.compose.project` on that same endpoint. Swarm or Kubernetes stack type → **400**. A file paste stack (no `GitConfig`, `WorkflowID` 0, `AutoUpdate` nil) continues. `GitConfig` set, `WorkflowID != 0`, or `AutoUpdate` set → **409**. Status already deploying → **409**. The stack file is not read in those cases.
5. Read the compose file Portainer stores (`ProjectPath` + `EntryPoint`). Patch with `stackutils.PatchServiceImageTag` using the compose service label as the service key. The edit is that service's `image:` line. The file is not remarshalled, so YAML anchors stay. After the edit, a re-parse refuses the result if any other service's image string changed or if the repository (everything before the tag) changed. The tag colon is the last `:` after the last `/` (`localhost:5000/app:1` has repository `localhost:5000/app` and tag `1`).
6. When step 5 succeeds and the file tag and the running image already equal `repository:version` → **200** `unchanged`.
7. `ImagePull` of `repository:version` on that same endpoint. `PullOptions` is empty: no registry login. Manifest unknown or a pull stream error → **422**. The stack file is not written. Compose up is not called.
8. If step 5 refused the patch (digest pin, duplicate `image:` lines, missing image, non-literal image, or a failed safety re-parse) and the file image still splits into a repository → **500** `Refused to patch the service image` after the pull. `repo:tag@sha256:…` is pulled as `repo:tag`. The stack file is not written. Compose up is not called. An image that does not split into a repository is **500** `Unable to read the service image` before the pull, and the file is still not written.
9. When the patched bytes differ, `StoreStackFileFromBytes` writes them under the decimal stack id and the stack entry point. That is the file Portainer will use on the next compose up. Git YAML is not updated.
10. `ComposeStackManager.Up` with exactly one service:

```go
err := manager.Up(ctx, stack, endpoint, portainer.ComposeUpOptions{
    ForceRecreate: false,
    Prune:         false,
    Services:      []string{serviceName},
})
```

A non-empty `Services` list keeps that service and drops its dependencies (`docker compose up --no-deps`). `ForceRecreate` and orphan removal stay off, so sibling containers are not force-recreated and are not deleted. This route never sends an empty service list. An empty list is the whole-project up used by other callers.

11. If compose up fails and this call wrote new bytes, the previous bytes are written back with the same `StoreStackFileFromBytes` call. `Down` is not called. Containers are not deleted. If the file was already on that tag, a failed up does not rewrite it.

A per-stack mutex on this route covers the re-read, patch, pull, write, and up. It does not lock `PUT /api/stacks/{id}`. Container listing happens before the lock.

Audit log (zerolog, message `stack container image update`): `endpointId`, `stackId`, `stackName`, `service`, `container`, `previousTag`, `newTag`, `status`. Never the token.

## Errors

Error bodies are Portainer's usual `{"message","details"}`. `details` is the underlying error with its first letter capitalized. It does not repeat the token.

| Status | When | `message` |
|---|---|---|
| 200 | Tag applied, or the one service was recreated | response `status` `updated` |
| 200 | File tag and running image already equal `repository:version` | response `status` `unchanged` |
| 400 | Tag is empty, or contains `/`, `:`, or whitespace | Invalid image tag |
| 400 | Swarm or Kubernetes stack | Only a compose stack can be updated |
| 400 | Container has no compose project or service label | Container is not a compose service |
| 400 | Service label is not in the stack file | Compose service is not in the stack file |
| 400 | Token matched and the body is invalid | Invalid request payload |
| 401 | Missing or wrong token | Invalid stack update token |
| 404 | No container matches | No container matches the requested name |
| 404 | Container matched but no stack with that project name on that environment | No compose stack matches the container |
| 409 | Two or more containers match | More than one container matches |
| 409 | Two stacks with that project name on the endpoint | More than one stack matches the container |
| 409 | Git-backed stack | Git-backed stack cannot be updated in place |
| 409 | Stack status is already deploying | Stack deployment is already in progress |
| 422 | Image pull failed | Failed to pull image |
| 500 | Patch refused | Refused to patch the service image |
| 500 | Compose up failed | Failed to update the container |
| 500 | Compose up failed and the restore write failed | Failed to update the container and restore the stack file |
| 503 | `PORTAINER_STACK_UPDATE_TOKEN` unset or empty | Stack update token is not configured |

There is no full-stack redeploy on any of these.

Other **500** messages, also with no successful update: `Unable to list environments`, `Unable to list containers on every environment`, `Unable to find the stack for the container`, `Stack file path is missing`, `Unable to read the stack file`, `Unable to patch the service image`, `Unable to read the service image`, `Unable to persist the stack file`, `Compose stack manager is not configured`.

A bad tag (**400**) and a missing service in the file (**400**) happen after lookup and after the file is read. They do not pull and they do not write. A pull failure (**422**) leaves the stored file unchanged. A refused patch whose image still has a repository (`repo:tag@sha256:…` and the other `ErrImageTagRefused` cases) attempts the pull first, then returns **500** with the file unchanged.

A private tag that needs registry auth returns **422** and does not change the stack file. A tag that exists for the wrong architecture can still pull. The endpoint does not choose a platform.

## Out of scope

- `PUT /api/stacks/{id}`, `POST /api/stacks/webhooks/{webhookID}`, and `POST /api/webhooks/{token}` (those redeploy a stack or recreate a container without this file patch).
- Writing git `yammy*.yaml` or any file under `docker-composes/production/`.
- Recreating sibling services, `compose down`, or prune.
- A Portainer UI, mail, or an agent in the request path.
- Updating the `portainer` container through this route.
