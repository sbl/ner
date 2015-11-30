#!/bin/sh

# install mitie for the CI environment
set -e

cd /tmp
git clone https://github.com/mit-nlp/MITIE.git
mkdir -p /tmp/mitie/include /tmp/mitie/lib
cd MITIE/mitielib && make && make install INSTALL_PREFIX=/tmp/mitie
