# docker build -t doctl_local --build-arg DOCTL_VERSION=1.23.1 .
#
# This Dockerfile builds in the first stage and adds it into a clean base in the second.
# If you want to develop in docker, please use `make docker_build` instead.
FROM golang:1.14.6-alpine3.11 AS build
WORKDIR /app
COPY . /app/
ARG GOARCH=""
RUN CGO_ENABLED=0 GOARCH="$GOARCH" go build \
  -a \
  -installsuffix cgo \
  -ldflags "-extldflags '-static' -s -w" \
  -o doctl \
  cmd/doctl/main.go

FROM alpine:3.11
RUN apk add --no-cache curl ca-certificates
RUN adduser -D user
WORKDIR /app
COPY --from=build /app/doctl /app/doctl
USER user
ENTRYPOINT ["/app/doctl"]
