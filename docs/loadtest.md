# Load test


### Example response

```shell
make load-test
Executando teste de carga... 
99th percentile: 9.478103ms
95th percentile: 3.454893ms
Mean: 1.923937ms
Max: 31.784834ms
Requests per second: 50.03
Success ratio: 100.00%
Status codes: map[202:1500]
Total requests: 1500

=== Relatório Detalhado ===
Requests      [total, rate, throughput]         1500, 50.03, 50.03
Duration      [total, attack, wait]             29.98s, 29.979s, 1.116ms
Latencies     [min, mean, 50, 90, 95, 99, max]  490.459µs, 1.924ms, 1.538ms, 2.847ms, 3.455ms, 9.478ms, 31.785ms
Bytes In      [total, mean]                     0, 0.00
Bytes Out     [total, mean]                     262134, 174.76
Success       [ratio]                           100.00%
Status Codes  [code:count]                      202:1500  
Error Set:

```


