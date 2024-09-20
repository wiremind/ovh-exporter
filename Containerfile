FROM golang:1.23-alpine AS build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

ARG JQ_VERSION=1.7

# hadolint ignore=DL3018
RUN apk update && apk add --no-cache bash git make binutils wget \
	&& wget --progress=dot:giga "https://github.com/jqlang/jq/releases/download/jq-${JQ_VERSION}/jq-${GOOS}-${GOARCH}" -O /usr/bin/jq \
	&& chmod +x /usr/bin/jq

WORKDIR $GOPATH/src/github.com/wiremind/ovh-exporter

COPY . .

RUN make ovh-exporter && mv ovh-exporter /usr/bin/


FROM busybox:stable AS runtime

COPY --from=build /usr/bin/ovh-exporter /usr/bin/ovh-exporter

ENTRYPOINT ["/usr/bin/ovh-exporter"]
