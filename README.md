# webhook

WebhookService comunicate internal services

mockgen -source=internal/infra/repository/adapter.go -destination=internal/infra/repository/repository_mock.go -package=repository


OK -> Criar o publisher (Expires, maxRetries, etc) 
Modificar o criador do externalEvent para salvar também o header
Modificar o gate para ler o header e passar postagem do hook
Criar testes
Adicionar logs
Filtros de mensagens publicadas