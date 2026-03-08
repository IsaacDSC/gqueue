#!/usr/bin/env bash

URL="http://localhost:8083/api/v1/task"
AUTH="Basic YWRtaW46cGFzc3dvcmQ="
DATA_FILE="example/task_event_payload.json"

while true; do
  # 10 chamadas em paralelo
  for i in {1..10}; do
    curl -s -X POST \
      "$URL" \
      -H "Content-Type: application/json" \
      -H "Authorization: $AUTH" \
      -d @"$DATA_FILE" &
  done

  # espera as 10 terminarem (opcional, mas recomendado)
  wait

  # espera 100ms
  sleep 1
done
