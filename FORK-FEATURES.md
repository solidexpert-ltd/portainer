# SolidExpert Portainer CE — fork features (Wave A+B)

Unofficial fork of [portainer/portainer](https://github.com/portainer/portainer). Not affiliated with Portainer Business Edition.

## Wave A (ops / packaging)

| Feature | How |
| --- | --- |
| **without-annoying UI** | Published image wraps CE with [ngxson/portainer-ce-without-annoying](https://github.com/ngxson/portainer-ce-without-annoying)-style Node proxy (`build/without-annoying/`). Hides BE upsell, empty MOTD, blocks Matomo. |
| **Scheduled Portainer backup** | Sidecar stack: `docker-composes/production/portainer/portainer-ce.stack.yml` using `dockurr/portainer-backup`. |

## Wave B (in-tree)

| Feature | Source |
| --- | --- |
| **CPU / memory / block I/O columns** on container list | Upstream PR [#13129](https://github.com/portainer/portainer/pull/13129) (cherry-picked) |
| **Stack & container webhooks + CE RBAC role UI unlock** | [quanla93/portainer](https://github.com/quanla93/portainer) |
| **S3-compatible backups in CE** (R2 / MinIO / S3 API) | quanla93 (`feat(backup): enable S3-compatible backups` + credential fix) |
| **Change Git repository URL for Docker stacks** | Already in upstream **2.45** via **Git Sources**: edit Source → Repository URL, or re-point the stack to another Source in *Edit Git settings*. PR [#12855](https://github.com/portainer/portainer/pull/12855) targets a pre-Sources UI and was **not** ported. |

## Images

- Docker Hub: `solidexpert/portainer-ce`
- GHCR: `ghcr.io/solidexpert-ltd/portainer-ce`

CI: `.github/workflows/build-docker.yml` builds CE binaries, packs `build/linux/Dockerfile`, then overlays `build/without-annoying/`.

## Deploy

See `../docker-composes/production/portainer/README.md` (monorepo) for the Portainer + backup compose paste.
