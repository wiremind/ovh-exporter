FROM golang:1.27.4@sha256:5e8afd493a7124a8586e134d9b532731161d4b9becf9dee5a5956b107426f93d AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /workspace

COPY . .

RUN make ovh-exporter && mv ovh-exporter /usr/bin/

# distroless/static brings the CA certificates the OVH, Cloudflare and RIPEstat
# HTTPS calls need, and no shell.
FROM gcr.io/distroless/static:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931 AS runtime

COPY --from=build /usr/bin/ovh-exporter /usr/bin/ovh-exporter
USER 65532:65532

ENTRYPOINT ["/usr/bin/ovh-exporter"]

CMD ["serve"]
