# webhook

# Documentação do Sistema de Webhook em Português

## Introdução

Este sistema de webhook é uma aplicação desenvolvida em Go que permite a comunicação entre serviços internos através de eventos. O projeto implementa uma arquitetura orientada a eventos (Event-Driven Architecture) que possibilita o registro, disparo e processamento de eventos entre diferentes serviços.

## Arquitetura do Sistema

O sistema é dividido em dois serviços principais:

1. **Servidor HTTP** - Responsável por expor endpoints que permitem criar eventos internos, registrar gatilhos (triggers) e publicar eventos externos.

2. **Worker** - Responsável pelo processamento assíncrono de tarefas, como o envio de webhooks para os serviços registrados.

### Tecnologias Utilizadas

- **Go 1.23.7** - Linguagem de programação principal
- **MongoDB** - Banco de dados para persistência de eventos e gatilhos
- **Redis** - Utilizado para gerenciamento de filas de tarefas assíncronas
- **Asynq** - Biblioteca para processamento de tarefas em background
- **Docker** - Contêinerização da aplicação

## Componentes Principais

### Eventos Internos

Os eventos internos são definidos pela estrutura `InternalEvent` e contêm:

- ID: Identificador único do evento (UUID)
- Nome: Nome do evento
- Nome do Serviço: Serviço que criou o evento
- URL do Repositório: Link para o repositório do serviço
- Equipe Responsável: Equipe responsável pelo serviço
- Gatilhos: Lista de serviços que serão notificados quando o evento ocorrer
- Timestamps: Data de criação, atualização e exclusão

### Gatilhos (Triggers)

Os gatilhos definem como um serviço deseja receber notificações de eventos. São compostos por:

- ID: Identificador único do gatilho (UUID)
- Nome do Serviço: Serviço que receberá a notificação
- Tipo: Tipo do gatilho (fireForGet, persistent, notPersistent)
- URL Base: Endereço base do serviço
- Caminho: Endpoint específico que receberá o webhook
- Timestamps: Data de criação, atualização e exclusão

### Tipos de Gatilho

- **fireForGet**: Dispara a notificação e não espera resposta
- **persistent**: Mantém a notificação persistente até que seja processada com sucesso
- **notPersistent**: Envia a notificação uma única vez, sem tentativas adicionais em caso de falha

## Fluxo de Funcionamento

1. Um serviço cria um evento interno via endpoint `/event/create`
2. Outros serviços podem se registrar para receber notificações deste evento via endpoint `/event/register`
3. Quando ocorre um evento externo, é enviado para o endpoint `/event/publisher`
4. O sistema verifica os serviços registrados para o evento e envia webhooks para cada um deles
5. O worker processa as tarefas de envio de webhooks de forma assíncrona, respeitando os tipos de gatilho configurados

## APIs Disponíveis

### Criação de Eventos Internos
- **Endpoint**: POST /event/create
- **Descrição**: Cria um novo evento interno no sistema

### Registro de Gatilhos
- **Endpoint**: POST /event/register
- **Descrição**: Registra um serviço para receber notificações de um evento específico

### Publicação de Eventos Externos
- **Endpoint**: POST /event/publisher
- **Descrição**: Publica um evento externo que será processado e enviado para os serviços registrados

## Configuração do Ambiente

O sistema utiliza Docker Compose para facilitar a execução em ambiente de desenvolvimento e produção. O arquivo `compose.yaml` define os seguintes serviços:

- **server**: Contêiner principal que executa o servidor e o worker
- **redis**: Serviço Redis para gerenciamento de filas
- **mongodb**: Banco de dados para persistência

### Variáveis de Ambiente

- **DB_CONNECTION_STRING**: String de conexão com o MongoDB
- **CACHE_ADDR**: Endereço do servidor Redis

### Recursos Alocados

O serviço principal tem as seguintes configurações de recursos:
- CPU: limite de 1 CPU, reserva de 0.5 CPU
- Memória: limite de 1GB, reserva de 512MB

## Executando o Sistema

### Iniciar todos os serviços
```bash
docker-compose up
```

### Iniciar apenas o servidor HTTP
```bash
go run . --service=webhook
```

### Iniciar apenas o worker
```bash
go run . --service=worker
```

### Iniciar ambos os serviços
```bash
go run .
# ou
go run . --service=all
```

## Teste de Carga

O sistema possui um módulo para testes de carga utilizando a biblioteca Vegeta, permitindo simular grande volume de requisições para validar a performance e escalabilidade da solução.

