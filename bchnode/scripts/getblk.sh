set -eux
curl -X POST --data "{\"method\":\"getblock.Call\",\"params\":[\"$1\"],\"id\":1}"  -H "Content-Type: text/plain" http://localhost:1234