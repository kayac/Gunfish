#!/bin/bash
set -eu

version='1.6.3'

if [ ! -d h2o-$version ] ; then
    wget https://github.com/h2o/h2o/archive/v$version.tar.gz
    tar xzf v$version.tar.gz
fi
cd h2o-$version

insert_num=$(grep -n MRuby misc/mruby_config.rb | awk -F':' '{print $1}')
insert_gem="conf.gem :git => 'https://github.com/matsumoto-r/mruby-sleep.git'"
sed -i "${insert_num} a ${insert_gem}" misc/mruby_config.rb

cmake -DWITH_BUNDLED_SSL=on .
make
sudo make install
