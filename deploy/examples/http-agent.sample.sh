# HTTP / CLI agent integration sample
# Set LAYA_HOST to the deploy machine (local: 127.0.0.1, LAN: your server IP).

export LAYA_HOST="127.0.0.1"
export LAYA_PORT="7710"
export LAYA_URL="http://${LAYA_HOST}:${LAYA_PORT}"

# health / models
curl -sS "${LAYA_URL}/health"
curl -sS "${LAYA_URL}/v1/models"

# native decide
curl -sS "${LAYA_URL}/v1/decide" \
  -H 'content-type: application/json' \
  -d '{"model":"laya-base","features":{"intent_change":0.9,"target_known":0.8},"top_k":3}'

# JEV-compatible decide
curl -sS "${LAYA_URL}/v1/jev/decide" \
  -H 'content-type: application/json' \
  -d '{"features":{"blocked":1,"ambiguity":0.8},"top_k":2}'

# CLI (if laya is on PATH)
# laya health --url "${LAYA_URL}"
# laya decide --url "${LAYA_URL}" --feature goal_done=1 --json
