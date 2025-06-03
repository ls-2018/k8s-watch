ARG BUILD_BASE_IMAGE=registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
ARG BASE_IMAGE=registry.cn-hangzhou.aliyuncs.com/acejilam/alpine
ARG BASE_IMAGE_VERSION=3.21@sha256:56fa17d2a7e7f168a043a2712e63aed1f8543aeafdcee47c58dcffe38ed51099

FROM --platform=$BUILDPLATFORM ${BUILD_BASE_IMAGE} AS builder
WORKDIR /data
COPY ./go.mod ./
COPY ./go.sum ./

RUN go mod download -x

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/.gopath \
    go build -o ./bin/watch .

ARG BASE_IMAGE
ARG BASE_IMAGE_VERSION
FROM ${BASE_IMAGE}:${BASE_IMAGE_VERSION}

WORKDIR /
COPY --from=builder /data/bin/watch /watch

RUN set -eux; \
    mkdir -p /log /tmp && \
    chown -R nobody:nobody /log && \
    chown -R nobody:nobody /tmp && \
    chown -R nobody:nobody /watch && \
    apk --no-cache --update upgrade && \
    apk --no-cache add ca-certificates && \
    apk --no-cache add tzdata && \
    rm -rf /var/cache/apk/* && \
    update-ca-certificates && \
    echo "only include root and nobody user" && \
    echo -e "root:x:0:0:root:/root:/bin/ash\nnobody:x:65534:65534:nobody:/:/sbin/nologin" | tee /etc/passwd && \
    echo -e "root:x:0:root\nnobody:x:65534:" | tee /etc/group && \
    rm -rf /usr/local/sbin/* && \
    rm -rf /usr/local/bin/* && \
    rm -rf /usr/sbin/* && \
    rm -rf /usr/bin/* && \
    rm -rf /sbin/* && \
    rm -rf /bin/*

ENTRYPOINT ["/watch"]