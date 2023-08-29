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
http  http://keycloak.test/api/dm/list?state=all Authorization:"Bearer $TOKEN"
http  "http://keycloak.test/api/dm/log?name=pippo&rows=8&state=all" Authorization:"Bearer $TOKEN" 
http  "http://keycloak.test/api/dm/searchbyname?name=pluto&state=all" Authorization:"Bearer $TOKEN" 
http --form POST "http://keycloak.test/api/dm/stopit?name=pippo"  Authorization:"Bearer $TOKEN" 
http --form POST "http://keycloak.test/api/dm/restart?name=pippo&state=all"  Authorization:"Bearer $TOKEN"
http  "http://keycloak.test/api/dm/listbyimage?imgname=nginx&state=all" Authorization:"Bearer $TOKEN"
http DELETE http://keycloak.test/api/dm/removexitbyimage?imgname=nginx Authorization:"Bearer $TOKEN" 
http --form POST  http://keycloak.test/api/dm/stopallcontainerbyimage?imgname=nginx  Authorization:"Bearer $TOKEN"
http  "http://keycloak.test/api/dm/exited"  Authorization:"Bearer $TOKEN"
http --form POST "http://keycloak.test/api/dm/create?name=ciccio&network=bridge&imgname=nginx"  Authorization:"Bearer $TOKEN"
http  "http://keycloak.test/api/dm/volumelist"  Authorization:"Bearer $TOKEN
http --form POST  http://keycloak.test/api/dm/volumecreate  Authorization:"Bearer $TOKEN"  <<< '{"Name":"VOLUME1"}'
http --form --json POST http://keycloak.test/api/dm/volumeprunes Authorization:"Bearer $TOKEN"
echo '{"Image":"nginx","Network":"bridge","MountPo":"/mnt","Volume":"ciro"}' | http --form POST http://keycloak.test/api/dm/create Authorization:"Bearer $TOKEN""
curl -X  POST http://keycloak.test/api/dm/create -H 'Content-Type: application/json' -d '{"Name":"pippo3","Image":"nginx","Network":"bridge","VolumeType":"Bind","MountPo":"/mnt","Volume":"ciro"}'  -H authorization:"Bearer $TOKEN"
http --form --json POST http://keycloak.test/api/dm/create Authorization:"Bearer $TOKEN"  <<< '{"Name":"davide1","Image":"nginx","Network":"bridge","VolumeType":"Bind","MountPo":"/mnt","Volume":"ciro"}'
 http --form --json POST http://keycloak.test/api/dm/create Authorization:"Bearer $TOKEN" <<< '{"Name":"davide3","Image":"nginx","Network":"bridge","Volumes":[{"VolumeType":"Bind","MountPo":"/mnt","Volume":"/mntbind"},{"VolumeType":"Volume","MountPo":"/mntciro","Volume":"ciro"}]}'


