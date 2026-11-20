FROM golang:1.27.4@sha256:205c7636d74e0ffbe3d007162df7d74f859f5a8c8322dfb9f907977686c126cf AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /workspace

COPY . .

RUN make ovh-exporter && mv ovh-exporter /usr/bin/

# distroless/static brings the CA certificates the OVH, Cloudflare and RIPEstat
# HTTPS calls need, and no shell.
FROM gcr.io/distroless/static:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329 AS runtime

COPY --from=build /usr/bin/ovh-exporter /usr/bin/ovh-exporter
USER 65532:65532

ENTRYPOINT ["/usr/bin/ovh-exporter"]

CMD ["serve"]
