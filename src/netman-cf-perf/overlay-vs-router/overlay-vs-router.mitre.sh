#!/bin/bash

OVERLAY_IPS=( "10.255.10.47:8080" "10.255.33.91:8080" "10.255.4.69:8080" "10.255.77.178:8080" )

curl -X DELETE proxy-1.mitre.c2c.cf-app.com/stats

for i in `seq 400`; do
  curl -s proxy-1.mitre.c2c.cf-app.com/proxy/proxy-2.mitre.c2c.cf-app.com > /dev/null
done

echo "stats through gorouter"
echo ""
curl proxy-1.mitre.c2c.cf-app.com/stats | jq .latency | tr -d ',' | tr -d ' ' | grep '\.' | tee mitre-router.txt

curl -X DELETE proxy-1.mitre.c2c.cf-app.com/stats

for i in `seq 100`; do
  for ip in "${OVERLAY_IPS[@]}"; do
    curl -s proxy-1.mitre.c2c.cf-app.com/proxy/"${ip}" > /dev/null
  done
done

echo "stats through overlay"
echo ""
curl proxy-1.mitre.c2c.cf-app.com/stats | jq .latency | tr -d ',' | tr -d ' ' | grep '\.' | tee mitre-overlay.txt

echo "stats also written to these files:"
echo "mitre-router.txt"
echo "mitre-overlay.txt"
