#!/usr/bin/env bash
set -e
# set -x

echo "=== PUT test data ==="

curl -X PUT -H "x-api-token: secret" http://127.0.0.1:9101/target -d @example-001.json
curl -X PUT -H "x-api-token: secret" http://127.0.0.1:9101/target -d @example-002.json

echo "=== GET test data ==="
curl -X GET -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d '{"id":"b055002b-c13a-56dd-81d0-4030e70b841b"}' | grep 'node'
curl -X GET -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d '{"id":"2ead9294-3095-511f-8d49-a4d972b73fba"}' | grep 'alertmanager'

echo "=== GET metrics data ==="
curl -X GET -s http://127.0.0.1:9101/metrics | grep 'inventor'

#echo "=== DELETE test data ==="
#curl -X DELETE -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d '{"id":"2ead9294-3095-511f-8d49-a4d972b73fba"}'
#curl -X GET -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d '{"id":"2ead9294-3095-511f-8d49-a4d972b73fba"}'| grep 'Id not found'

echo "Done"
