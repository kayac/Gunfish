GIT_VER:=$(shell git describe --tags)
DATE:=$(shell date +%Y-%m-%dT%H:%M:%SZ)
export GO111MODULE:=on

.PHONY: test install clean

all: test

install:
	 cd cmd/gunfish && go build -ldflags "-X main.version=${GIT_VER} -X main.buildDate=${DATE}"
		install cmd/gunfish/gunfish ${GOPATH}/bin

packages:
	cd cmd/gunfish \
		&& CGO_ENABLED=0 gox \
			-os="linux darwin" \
			-arch="amd64" \
			-output "../../pkg/{{.Dir}}-${GIT_VER}-{{.OS}}-{{.Arch}}" \
			-gcflags "-trimpath=${GOPATH}" \
			-ldflags "-w -X main.version=${GIT_VER} -X main.buildDate=${DATE} -extldflags \"-static\"" \
			-tags "netgo"
	cd pkg && find . -name "*${GIT_VER}*" -type f -exec zip {}.zip {} \;

gen-cert:
	test/scripts/gen_test_cert.sh

test: gen-cert
	go test -v ./apns
	go test -v ./fcm
	go test -v .

clean:
	rm -f cmd/gunfish/gunfish
	rm -f test/server.*
	rm -f pkg/*

build:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" cmd/gunfish/gunfish.go

tools/%:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" test/tools/$*/$*.go
