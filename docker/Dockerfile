FROM alpine:3.24
LABEL maintainer="FUJIWARA Shunichiro <fujiwara.shunichiro@gmail.com>"

ARG VERSION
ARG TARGETPLATFORM
COPY dist/docker/${TARGETPLATFORM}/gunfish /usr/local/bin/gunfish
EXPOSE 8003
WORKDIR /opt/gunfish

ENTRYPOINT ["/usr/local/bin/gunfish"]
