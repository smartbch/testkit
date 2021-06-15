set -eux
curl -X POST --data "{\"method\":\"internal\",\"params\":[$1],\"id\":1}"  -H "Content-Type: text/plain" http://localhost:1234