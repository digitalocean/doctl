# docker build -t doctl_local --build-arg DOCTL_VERSION=1.23.1 .
#
# This Dockerfile exists so casual uses of `docker build` and `docker run` do something sane.
# We don't recommend using it: If you want to develop in docker, please use `make docker_build`
# instead.

FROM alpine:3.8

ARG DOCTL_VERSION
ENV DOCTL_VERSION=$DOCTL_VERSION

RUN apk add --no-cache curl

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

WORKDIR /app

RUN curl -L https://github.com/digitalocean/doctl/releases/download/v${DOCTL_VERSION}/doctl-${DOCTL_VERSION}-linux-amd64.tar.gz  | tar xz

ENTRYPOINT ["./doctl"]
