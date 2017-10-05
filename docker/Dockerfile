FROM alpine:3.6
MAINTAINER Fujiwara Shunichiro <fujiwara.shunichiro@gmail.com>

RUN apk --no-cache add unzip curl && \
    mkdir -p /etc/gunfish /opt/gunfish && \
    curl -sL https://github.com/kayac/Gunfish/releases/download/v0.2.4/gunfish-v0.2.4-linux-amd64.zip > /tmp/gunfish-v0.2.4-linux-amd64.zip && \
    cd /tmp && \
    unzip gunfish-v0.2.4-linux-amd64.zip && \
    install gunfish-v0.2.4-linux-amd64 /usr/bin/gunfish && \
    rm -f /tmp/gunfish*

WORKDIR /opt/gunfish

ENTRYPOINT ["/usr/bin/gunfish"]
