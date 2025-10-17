# Plan Charge tests and requirements

## Objetive

- Validate performance of receive multiples simultaneously messages in 5 minutes
- Validate performance of delivery multiple messages for 15 consumers
  - using google pubsub
  - using redis asynq

### Metrics

- Total time consumer all msgs
- Time max 5min producer simultaneously messages
- Efectivity (Success/Failures) producer
- Efectivity (Success/Failures) consumer
- Total de retries Producer/Consumer

### Steps requirements to tests cases

1. Create events
   1.1 user.created
   1.2 payment.created

2. Create consumers
   2.1 user.created
   2.2 payment.created

### Test Scenarios

3. Publisher events in parallel at 5min
   3.1 Create users in parallel
   3.2 Create payments in parallel
   3.3 Agregate in counter ++

4. Consumers messages in parallel at 7min
   4.1 Agregate in counter ++
