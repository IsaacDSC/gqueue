# Usage

## Start Gqueue services

open the terminal and run this command:
```sh
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile observability up -d --build
```

## Start consumer example

open new tab terminal and run this commando to starting example consumer application

```sh
docker compose -f deployment/app-pgsql/docker-compose.yaml --profile example up
```

## Register new consumer

open new tab terminal and execute calling to backoffice service to register new consumer
```sh
curl -X PUT \
 http://localhost:8080/api/v1/event/consumer \
 -H "Content-Type: application/json" \
 -H "Accept: application/json" \
 -H "Authorization: Basic YWRtaW46cGFzc3dvcmQ=" \
 -d @example/event_data.json
```

## Simulate charge publisher message

*Execute this command generate charge publisher message*

#### Using pubsub
```
sh ./example/simulation/multiples_producer_pubsub.sh
```

#### Using task
```
sh ./example/simulation/multiples_producer_task.sh

```

## Open your browser

- [Dashboard-Pubsub](http://localhost:3000/d/adhqlpf/gqueue-pubsub-service-dashboard?orgId=1&from=now-3h&to=now&timezone=browser&refresh=5s)
- [Dashboard-Task](http://localhost:3000/d/adxctbttask/gqueue-task-service-dashboard?orgId=1&from=now-15m&to=now&timezone=browser&refresh=5s)
- [Dashboard-Backoffice](http://localhost:3000/d/adxctbtbackoffice/gqueue-backoffice-service-dashboard?orgId=1&from=now-3h&to=now&timezone=browser&refresh=5s)
