# Overlay based on ngxson/portainer-ce-without-annoying:
# wraps a Portainer CE image with a tiny Node proxy that strips BE upsell UI
# and blocks Matomo tracking, while keeping the CE binary intact.
ARG BASE_IMAGE=solidexpert/portainer-ce:latest
FROM ${BASE_IMAGE} AS portainer

FROM node:22.23.1-alpine3.24

WORKDIR /
COPY --from=portainer . .

WORKDIR /proxy
RUN npm i express@^4 http-proxy-middleware@^2
COPY app.js .

COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh

EXPOSE 9000 9443 8000
VOLUME /data

ENTRYPOINT ["/docker-entrypoint.sh"]
