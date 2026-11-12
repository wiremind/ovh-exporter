FROM golang:1.27.0@sha256:4013ae0f9e7994f8535c58c811f8f863fbed38b72e0d51e6592156f758d66146 AS build

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
