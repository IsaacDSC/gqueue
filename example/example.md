### Example create event

curl -X POST \
 http://localhost:8080/api/v1/event/consumer \
 -H "Content-Type: application/json" \
 -H "Accept: application/json" \
 -d @example/event_data.json

### Example get event

curl -X GET \
 http://localhost:8080/api/v1/my-app/events/payment.processed | jq

curl -X GET \
 http://localhost:8080/api/v1/events

curl -X GET \
 'http://localhost:8080/api/v1/events?team_owner=my-team'

### Publisher data

curl -X POST \
 http://localhost:8080/api/v1/event/publisher \
 -H "Content-Type: application/json" \
-d @example/publisher_data.json

### DELETE event

curl -i -X DELETE \\n"http://localhost:8080/api/v1/event/da4543c5-3cca-4151-8737-5f4cf7fa702f"

### GET insights

curl -X GET \
http://localhost:8080/api/v1/insights | jq

### PATCH event

curl -X PATCH \
 'http://localhost:8080/api/v1/event/da4543c5-3cca-4151-8737-5f4cf7fa702f' \
 -H "Content-Type: application/json" \
 -H "Accept: application/json" \
 -d @example/path_event_data.json
