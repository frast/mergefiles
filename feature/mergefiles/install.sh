#!/bin/sh
set -e

echo "Activating feature 'mergefiles'"

releaseVersion=${RELEASEVERSION:-undefined}

[ $(uname -m) = x86_64 ] && curl -L https://github.com/frast/mergefiles/releases/download/$releaseVersion/mergefiles_$releaseVersion_linux_amd64.tar.gz | tar xz mergefiles
[ $(uname -m) = aarch64 ] && curl -L https://github.com/cnoe-io/cnoe-cli/releases/download/$releaseVersion/mergefiles_$releaseVersion_linux_arm64.tar.gz | tar xz mergefiles
chmod +x ./mergefiles
mv ./mergefiles /usr/local/bin/mergefiles
