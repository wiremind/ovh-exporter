FROM golang:1.23-alpine AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# hadolint ignore=DL3018
RUN apk update \
	&& apk add --no-cache \
	make binutils

WORKDIR $GOPATH/src/github.com/wiremind/ovh-exporter

COPY . .

RUN make ovh-exporter && mv ovh-exporter /usr/bin/

FROM busybox:stable AS runtime

COPY --from=build /usr/bin/ovh-exporter /usr/bin/ovh-exporter

ENTRYPOINT ["/usr/bin/ovh-exporter"]

CMD ["serve"]
