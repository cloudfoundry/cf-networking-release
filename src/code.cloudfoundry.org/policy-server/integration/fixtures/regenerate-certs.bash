#!/bin/bash

set -e

this_dir="$(cd $(dirname $0) && pwd)"

pushd "$this_dir" > /dev/null

rm -rf out
# CA to distribute to cf-networking policy server and client
certstrap init --passphrase '' --common-name netman-ca

# Server certificate for the policy server
certstrap request-cert --passphrase '' --common-name policy-server.service.cf.internal --domain '*.policy-server.service.cf.internal,policy-server.service.cf.internal' --ip 127.0.0.1
certstrap sign policy-server.service.cf.internal --CA netman-ca
mv -f out/policy-server.service.cf.internal.key server.key
mv -f out/policy-server.service.cf.internal.crt server.crt

# Client certificate for the policy agent
certstrap request-cert --passphrase '' --common-name 'policy-agent'
certstrap sign policy-agent --CA netman-ca
mv -f out/policy-agent.key client.key
mv -f out/policy-agent.crt client.crt

mv -f out/netman-ca.{crl,crt,key} .

rm -rf out

popd > /dev/null

