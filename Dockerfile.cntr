FROM doctl_builder

ENV CGO=0
ENV OUT_D /out

RUN mkdir -p /out
RUN mkdir -p /go/src/github.com/digitalocean/doctl
ADD . /go/src/github.com/digitalocean/doctl/

RUN cd /go/src/github.com/digitalocean/doctl && \
    make build_linux_amd64 OUT=/out && \
    make build_linux_386   OUT=/out && \
    make build_mac         OUT=/out

RUN find /out

FROM alpine:latest

VOLUME ["/copy"]

RUN  apk update && \
     apk add rsync && \
     apk add libc6-compat

COPY --from=0 /out/* /app/

ENTRYPOINT ["/app/doctl_linux_amd64"]
