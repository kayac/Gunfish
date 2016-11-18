#!/bin/bash
set -uex

version=1.11.2.1
prefix=`git rev-parse --show-toplevel`/test/nginx

wget "https://openresty.org/download/openresty-$version.tar.gz"
tar -xzf openresty-$version.tar.gz
cd openresty-$version

./configure --with-http_v2_module --prefix=${prefix}
make
make install

cd ../
rm openresty-$version.tar.gz
rm -rf openresty-$version
