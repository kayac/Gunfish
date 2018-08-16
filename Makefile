GIT_VER:=$(shell git describe --tags)
DATE:=$(shell date +%Y-%m-%dT%H:%M:%SZ)

.PHONY: test get-deps install clean apnscli

all: test

install:
	 cd cmd/gunfish && go build -ldflags "-X main.version=${GIT_VER} -X main.buildDate=${DATE}"
		install cmd/gunfish/gunfish ${GOPATH}/bin

get-dep-on-ci:
	curl -sL https://github.com/golang/dep/releases/download/v0.3.1/dep-linux-amd64 > ${GOPATH}/bin/dep
	chmod +x ${GOPATH}/bin/dep

get-deps:
	dep ensure

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
	nohup h2o -c config/h2o/h2o.conf > h2o_access.log &
	go test -v ./apns || ( pkill h2o && exit 1 )
	go test -v ./fcm || ( pkill h2o && exit 1 )
	go test -v . || ( pkill h2o && exit 1 )
	pkill h2o

clean:
	rm -f cmd/gunfish/gunfish
	rm -f test/server.*
	rm -f pkg/*

build:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" cmd/gunfish/gunfish.go

apnscli:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" test/tools/apnscli/apnscli.go
