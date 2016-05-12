#!/bin/bash
set -e
script_path=$(dirname $(readlink -f $0))
gen_path=$script_path/..
#------------------------------------------------------------
# Creates secrete key
#------------------------------------------------------------
rm -rf $gen_path/server*
openssl genrsa 2048 > $gen_path/server.key

#------------------------------------------------------------
# Country Name (2 letter code) [AU]:
# State or Province Name (full name) [Some-State]:
# Locality Name (eg, city) []:
# Organization Name (eg, company) [Internet Widgits Pty Ltd]:
# Organizational Unit Name (eg, section) []:
# Common Name (e.g. server FQDN or YOUR name) []:
# Email Address []:
# 
# Please enter the following 'extra' attributes
# to be sent with your certificate request
# A challenge password []:
# An optional company name []:
#------------------------------------------------------------
openssl req -new -key $gen_path/server.key <<EOF > $gen_path/server.csr
JP
Kanagawa
Test Town
Test Company
Test Section
localhost



EOF

#------------------------------------------------------------
# Creates server certification
#------------------------------------------------------------
openssl x509 -days 3650 -req -signkey $gen_path/server.key < $gen_path/server.csr > $gen_path/server.crt
