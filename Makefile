PLATFORMS ?= linux/amd64,linux/arm64,linux/ppc64le
IMG ?= registry.cn-hangzhou.aliyuncs.com/ls-2018/k8s-watch-server:latest
GIT_TAG := $(shell git describe --tags --abbrev=0)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_STATE := $(shell if [ -z "$$(git status --porcelain)" ]; then echo "clean"; else echo "dirty"; fi)

LDFLAGS ?= -s -w
LDFLAGS += -X github.com/ls-2018/k8s-watch/cmd.progVersion=$(GIT_TAG)
LDFLAGS += -X github.com/ls-2018/k8s-watch/cmd.progCommit=$(GIT_COMMIT)
LDFLAGS += -X github.com/ls-2018/k8s-watch/cmd.progStatus=$(GIT_STATE)

all:
	docker build -t registry.cn-hangzhou.aliyuncs.com/ls-2018/k8s-watch-server:latest .
	docker push registry.cn-hangzhou.aliyuncs.com/ls-2018/k8s-watch-server:latest

docker-multiarch:
	docker buildx build -f ./Dockerfile --pull --no-cache --platform=$(PLATFORMS) --push . -t $(IMG)

deploy:
	kind load docker-image -n koord $(IMG)

build:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ./bin/k8s-watch-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o ./bin/k8s-watch-linux-arm64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ./bin/k8s-watch-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o ./bin/k8s-watch-darwin-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o ./bin/k8s-watch-windows-amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o ./bin/k8s-watch-windows-arm64.exe .