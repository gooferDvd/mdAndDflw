#!/bin/bash

client_id="docker-manager"
client_secret="v6MAf0wez1tztEK11rJDKetl9xHC5aVm"
username="admin"
password="admin"
host="http://keycloak.test"
realm="docker-test"

echo $client_id

TOKEN=$(curl -L -X POST $host/realms/$realm/protocol/openid-connect/token \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode client_id=$client_id \
--data-urlencode grant_type=password \
--data-urlencode client_secret=$client_secret \
--data-urlencode 'scope=openid' \
--data-urlencode username=$username \
--data-urlencode password=$password \
| jq -r '.access_token')

echo "$TOKEN"
http  http://keycloak.test/api/dm/getbindsrfl Authorization:"Bearer $TOKEN"
