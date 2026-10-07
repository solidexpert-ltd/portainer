# without-annoying overlay

Adapted from [ngxson/portainer-ce-without-annoying](https://github.com/ngxson/portainer-ce-without-annoying) (MIT).

The published `solidexpert/portainer-ce` / `ghcr.io/.../portainer-ce` image is this overlay on top of the SolidExpert Portainer CE build. It:

- hides Business Edition upsell UI chrome
- returns empty `/api/motd`
- blocks Matomo tracking script injection
