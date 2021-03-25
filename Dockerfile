# Step 1: Build
FROM golang:1.16-alpine AS build

ARG GOARCH=amd64
ENV OUT_D /out

RUN mkdir -p /out
RUN  apk add --update  --no-cache \
     bash \
     coreutils \
     git \
     libc6-compat \
     make

RUN mkdir -p /go/src/github.com/digitalocean/doctl
ADD . /go/src/github.com/digitalocean/doctl/

RUN cd /go/src/github.com/digitalocean/doctl && \
    make build GOARCH=$GOARCH

# Step 2: App
FROM alpine:3.12

RUN apk add --update --no-cache \
        ca-certificates \
        libc6-compat \
        openssh

WORKDIR /app
COPY --from=build /out/doctl /app/doctl

RUN adduser -D user
USER user:user

ENTRYPOINT ["/app/doctl"]
