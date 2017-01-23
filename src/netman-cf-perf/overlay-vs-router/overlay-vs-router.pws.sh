#!/bin/bash

curl -X DELETE proxy-1.cfapps.io/stats

echo "proxying requests through gorouter"
echo ""
for i in `seq 10000`; do
  curl -s proxy-1.cfapps.io/proxy/proxy-2.cfapps.io > /dev/null
done

echo "getting stats through gorouter"
echo ""
curl proxy-1.cfapps.io/stats | jq .latency | tr -d ',' | tr -d ' ' | grep '\.' > pws-router.txt

curl -X DELETE proxy-1.cfapps.io/stats

echo "discovering overlay ip addresses"
echo ""
OVERLAY_IPS=($(go run main.go))

echo "proxying requests through overlay"
echo ""
for i in `seq 100`; do
  for ip in "${OVERLAY_IPS[@]}"; do
    curl -s proxy-1.cfapps.io/proxy/"${ip}" > /dev/null
  done
done

echo "getting stats through overlay"
echo ""
curl proxy-1.cfapps.io/stats | jq .latency | tr -d ',' | tr -d ' ' | grep '\.' > pws-overlay.txt

echo "stats written to these files:"
echo "pws-router.txt"
echo "pws-overlay.txt"
