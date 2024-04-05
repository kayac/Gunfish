GIT_VER:=$(shell git describe --tags)
DATE:=$(shell date +%Y-%m-%dT%H:%M:%SZ)
export GO111MODULE:=on
export PROJECT_ROOT:=$(shell git rev-parse --show-toplevel)

.PHONY: test install clean

all: test

install:
	cd cmd/gunfish && go build -ldflags "-X main.version=${GIT_VER} -X main.buildDate=${DATE}"
		install cmd/gunfish/gunfish ${GOPATH}/bin

gen-cert:
	test/scripts/gen_test_cert.sh

test: gen-cert
	go test -v ./...

clean:
	rm -f cmd/gunfish/gunfish
	rm -f test/server.*
	rm -f dist/*

packages:
	goreleaser build --skip-validate --rm-dist

build:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" cmd/gunfish/gunfish.go

tools/%:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" test/tools/$*/$*.go

docker-build: # clean packages
		mv dist/Gunfish_linux_amd64_v1 dist/Gunfish_linux_amd64
		docker buildx build \
				--build-arg VERSION=${GIT_VER} \
				--platform linux/amd64,linux/arm64 \
				-f docker/Dockerfile \
				-t kayac/gunfish:${GIT_VER} \
				-t ghcr.io/kayac/gunfish:${GIT_VER} \
				.

docker-push:
		mv dist/Gunfish_linux_amd64_v1 dist/Gunfish_linux_amd64
		docker buildx build \
				--build-arg VERSION=${GIT_VER} \
				--platform linux/amd64,linux/arm64 \
				-f docker/Dockerfile \
				-t kayac/gunfish:${GIT_VER} \
				-t ghcr.io/kayac/gunfish:${GIT_VER} \
				--push \
				.
