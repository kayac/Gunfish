GIT_VER:=$(shell git describe --tags)
DATE:=$(shell date +%Y-%m-%dT%H:%M:%SZ)
export PROJECT_ROOT:=$(shell git rev-parse --show-toplevel)

.PHONY: test install clean packages build docker-prepare docker-build docker-push

all: test

install:
	go install -ldflags "-X main.version=${GIT_VER} -X main.buildDate=${DATE}" ./cmd/gunfish

gen-cert:
	test/scripts/gen_test_cert.sh

test: gen-cert
	go test -race -v ./...

clean:
	rm -f cmd/gunfish/gunfish
	rm -f test/server.*
	rm -f dist/*

packages:
	goreleaser build --skip=validate --clean

build:
	go build -trimpath -ldflags="-w" ./cmd/gunfish

tools/%:
	go build -trimpath -ldflags="-w" ./test/tools/$*

# Copy binaries built by goreleaser to the paths referred by docker/Dockerfile.
docker-prepare:
	rm -rf dist/docker
	install -D dist/Gunfish_linux_amd64_v1/gunfish dist/docker/linux/amd64/gunfish
	install -D dist/Gunfish_linux_arm64_v8.0/gunfish dist/docker/linux/arm64/gunfish

docker-build: docker-prepare
	docker buildx build \
		--build-arg VERSION=${GIT_VER} \
		--platform linux/amd64,linux/arm64 \
		-f docker/Dockerfile \
		-t ghcr.io/kayac/gunfish:${GIT_VER} \
		.

docker-push: docker-prepare
	docker buildx build \
		--build-arg VERSION=${GIT_VER} \
		--platform linux/amd64,linux/arm64 \
		-f docker/Dockerfile \
		-t ghcr.io/kayac/gunfish:${GIT_VER} \
		--push \
		.
