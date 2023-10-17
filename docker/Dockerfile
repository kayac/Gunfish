FROM alpine:3.18
LABEL maintainer "FUJIWARA Shunichiro <fujiwara.shunichiro@gmail.com>"

ARG VERSION
ARG TARGETARCH
ADD dist/Gunfish_linux_${TARGETARCH}/gunfish /usr/local/bin/gunifish
EXPOSE 8003
WORKDIR /opt/gunfish

ENTRYPOINT ["/usr/local/bin/gunfish"]
