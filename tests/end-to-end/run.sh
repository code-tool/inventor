#!/usr/bin/env bash
set -e
# set -x

# put_target <file> - registers a target and prints the id returned by the server.
# The id is read from the response rather than hardcoded, so the test doesn't
# break when the UUID derivation scheme changes (unit tests cover that).
put_target() {
  local id
  id=$(curl -X PUT -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d @"$1" \
    | sed -n 's/.*"id":"\([0-9a-f-]\{36\}\)".*/\1/p')
  if [ -z "$id" ]; then
    echo "PUT $1 did not return an id" >&2
    exit 1
  fi
  echo "$id"
}

echo "=== PUT test data ==="

ID1=$(put_target example-001.json)
ID2=$(put_target example-002.json)
echo "example-001.json -> $ID1"
echo "example-002.json -> $ID2"

echo "=== GET test data ==="
curl -X GET -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d "{\"id\":\"$ID1\"}" | grep 'smartctl'
curl -X GET -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d "{\"id\":\"$ID2\"}" | grep 'ipmi'

echo "=== GET metrics data ==="
curl -X GET -s http://127.0.0.1:9101/metrics | grep 'inventor'

#echo "=== DELETE test data ==="
#curl -X DELETE -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d "{\"id\":\"$ID2\"}"
#curl -X GET -H "x-api-token: secret" -s http://127.0.0.1:9101/target -d "{\"id\":\"$ID2\"}" | grep 'Id not found'

echo "Done"
