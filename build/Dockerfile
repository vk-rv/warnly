# https://goreleaser.com/docker/

FROM alpine:3.22.2

ARG BINARY=warnly

LABEL maintainer="dev@warnly.io"
LABEL org.opencontainers.image.name="warnly"
LABEL org.opencontainers.image.source="https://github.com/vk-rv/warnly"

RUN mkdir -p /etc/warnly && \
    mkdir -p /var/opt/warnly

COPY $BINARY /

RUN addgroup warnly && \
    adduser -S -D -g '' -G warnly -s /bin/sh warnly && \
    chown -R warnly:warnly /etc/warnly /var/opt/warnly

EXPOSE 8080
EXPOSE 8081

USER warnly

CMD ["./warnly"]
