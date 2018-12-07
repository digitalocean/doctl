FROM golang:1.11-alpine

ENV MAKE_TARGET linux_amd64

RUN  apk update && \
     apk add bash && \
     apk add curl && \
     apk add git && \
     apk add make && \
     apk add libc6-compat
