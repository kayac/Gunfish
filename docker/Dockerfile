FROM alpine:3.6
MAINTAINER Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

ARG version

RUN apk --no-cache add unzip curl && \
    mkdir -p /etc/gunfish /opt/gunfish && \
    curl -sL https://github.com/kayac/Gunfish/releases/download/${version}/gunfish-${version}-linux-amd64.zip > /tmp/gunfish-${version}-linux-amd64.zip && \
    cd /tmp && \
    unzip gunfish-${version}-linux-amd64.zip && \
    install gunfish-${version}-linux-amd64 /usr/bin/gunfish && \
    rm -f /tmp/gunfish*

WORKDIR /opt/gunfish

ENTRYPOINT ["/usr/bin/gunfish"]
