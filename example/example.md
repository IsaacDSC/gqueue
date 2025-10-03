### Example create event

curl -X POST \
 http://localhost:8080/event/consumer \
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

http://localhost:8080/api/v1/event/publisher \
-H "Content-type: application-json" \
-d @example/publisher_data.json
