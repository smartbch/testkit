set -eux
#pubkey_hex_string-voting_power_string-add or modify or delete
curl -X POST --data "{\"method\":\"cc\",\"params\":[\"$1\"],\"id\":1}"  -H "Content-Type: text/plain" http://localhost:1234