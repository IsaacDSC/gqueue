# limit configurations

> Warning about limits timeouts and rate limits


### Producer limits

https://github.com/IsaacDSC/gqueue/blob/03fc463d0d9f9a3ea3b37f0202708c3807e1503f/pkg/httpclient/client_http_with_logger.go#L93

```go 
client, err := clienthttp.New("",
    clienthttp.WithTimeout(30*time.Second),
    clienthttp.WithMaxIdleConns(100),
    clienthttp.WithMaxIdleConnsPerHost(2),
    clienthttp.WithIdleConnTimeout(90*time.Second),
    clienthttp.WithAuditor(auditor),
)
```

#### Pubsub
- 200ms timeout response producer message
- 2000 RPS

#### Task
- 200ms timeout response producer message
- 15 RPS


### Consumer limits

https://github.com/IsaacDSC/gqueue/blob/03fc463d0d9f9a3ea3b37f0202708c3807e1503f/cmd/setup/pubsub/http_api.go#L40

```go
server := &http.Server{
    Addr:         port.String(),
    Handler:      handler,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 200 * time.Millisecond,
    IdleTimeout:  60 * time.Second,
}

```

#### Pubsub
- 200ms timeout response producer message
- 2000 RPS

#### Task
- 200ms timeout response producer message
- ACK | NACK timeout 5m 
- 15 RPS