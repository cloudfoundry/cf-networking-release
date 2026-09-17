#!/bin/bash

set -e

this_dir="$(cd $(dirname $0) && pwd)"

pushd "$this_dir" > /dev/null

rm -rf out
certstrap init --common-name "CA" --passphrase ""
certstrap request-cert --common-name "client" --domain "client" --passphrase ""
certstrap sign client --CA "CA"

certstrap request-cert --common-name "metron" --domain "metron" --passphrase ""
certstrap sign metron --CA "CA"

mv -f out/* ./
rm -rf out

popd > /dev/null
