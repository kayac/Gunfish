GIT_VER:=$(shell git describe --tags)
DATE:=$(shell date +%Y-%m-%dT%H:%M:%SZ)

.PHONY: test get-deps install clean

all: test

install:
	 cd cmd/gunfish && go build -ldflags "-X=main.version ${GIT_VER} -X main.buildDate ${DATE}"
		install cmd/gunfish/gunfish ${GOPATH}/bin

get-deps:
	go get -t -d -v .
	cd cmd/gunfish && go get -t -d -v .

packages:
	cd cmd/gunfish && gox -os="linux darwin" -arch="amd64" -output "../../pkg/{{.Dir}}-${GIT_VER}-{{.OS}}-{{.Arch}}" -gcflags "-trimpath=${GOPATH}" -ldflags "-w -X main.version ${GIT_VER} -X main.buildDate ${DATE}"
	cd pkg && find . -name "*${GIT_VER}*" -type f -exec zip {}.zip {} \;

test: gen-cert build_nginx
	test/nginx/nginx/sbin/nginx -c conf/nginx.conf
	go test -v
	pkill nginx

gen-cert:
	test/scripts/gen_test_cert.sh

build_nginx:
	test/scripts/build_nginx.sh
	rm -f test/nginx/nginx/conf/nginx.conf
	cp conf/nginx.conf.example test/nginx/nginx/conf/nginx.conf

clean:
	rm -f cmd/gunfish/gunfish
	rm -f test/server.*
	rm -f pkg/*

build:
	go build -gcflags="-trimpath=${HOME}" -ldflags="-w" cmd/gunfish/gunfish.go
