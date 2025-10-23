### Example create event

curl -X POST \
 http://localhost:8080/api/v1/event/consumer \
 -H "Content-Type: application/json" \
 -H "Accept: application/json" \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
 -d @example/event_data.json

### Example get event

curl -X GET \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
 http://localhost:8080/api/v1/my-app/events/payment.processed | jq

curl -X GET \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
 http://localhost:8080/api/v1/events

curl -X GET \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
 'http://localhost:8080/api/v1/events?team_owner=my-team'

### Publisher data

curl -X POST \
 http://localhost:8080/api/v1/event/publisher \
 -H "Content-Type: application/json" \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
-d @example/publisher_data.json

### DELETE event

curl -i -X DELETE \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
"http://localhost:8080/api/v1/event/da4543c5-3cca-4151-8737-5f4cf7fa702f"

### GET insights

curl -X GET \
-H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
http://localhost:8080/api/v1/insights | jq

### PATCH event

curl -X PATCH \
 'http://localhost:8080/api/v1/event/da4543c5-3cca-4151-8737-5f4cf7fa702f' \
 -H "Content-Type: application/json" \
 -H "Accept: application/json" \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
 -d @example/path_event_data.json
