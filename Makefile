# Makefile para o projeto webhook

# Variáveis
GO=go
APP_NAME=webhook
PORT=8080

# Cores para output
GREEN=\033[0;32m
BLUE=\033[0;34m
YELLOW=\033[0;33m
NC=\033[0m # No Color

# Comandos principais
.PHONY: all build run test clean load-test run-worker run-webhook run-all

# Comandos por padrão
all: help

# Construir o aplicativo
build:
	@echo "$(GREEN)Construindo aplicação...$(NC)"
	@$(GO) build -o $(APP_NAME) ./cmd/main.go

# Rodar teste de carga
load-test:
	@echo "$(YELLOW)Executando teste de carga...$(NC)"
	@$(GO) run ./cmd/loadtest.go

# Iniciar worker
run-worker:
	@echo "$(BLUE)Iniciando serviço worker...$(NC)"
	@$(GO) run ./cmd/main.go --service=worker

# Iniciar webhook (API)
run-webhook:
	@echo "$(BLUE)Iniciando serviço webhook (API)...$(NC)"
	@$(GO) run ./cmd/main.go --service=webhook

# Iniciar ambos serviços
run-all:
	@echo "$(BLUE)Iniciando todos os serviços (worker e webhook)...$(NC)"
	@$(GO) run ./cmd/main.go --service=all

# Limpar binários gerados
clean:
	@echo "$(GREEN)Limpando binários...$(NC)"
	@rm -f $(APP_NAME)

# Executar testes
test:
	@echo "$(GREEN)Executando testes...$(NC)"
	@$(GO) test ./... -v

# Docker
docker-build:
	@echo "$(GREEN)Construindo imagem Docker...$(NC)"
	@docker build -t $(APP_NAME) .

docker-up:
	@echo "$(GREEN)Iniciando serviços com Docker Compose...$(NC)"
	@docker-compose up -d

docker-down:
	@echo "$(GREEN)Parando serviços do Docker Compose...$(NC)"
	@docker-compose down

# Ajuda
help:
	@echo "$(YELLOW)Comandos disponíveis:$(NC)"
	@echo "  $(GREEN)make build$(NC)        - Constrói a aplicação"
	@echo "  $(GREEN)make run-worker$(NC)   - Executa o serviço worker"
	@echo "  $(GREEN)make run-webhook$(NC)  - Executa o serviço webhook (API)"
	@echo "  $(GREEN)make run-all$(NC)      - Executa ambos os serviços"
	@echo "  $(GREEN)make load-test$(NC)    - Executa teste de carga"
	@echo "  $(GREEN)make test$(NC)         - Executa os testes"
	@echo "  $(GREEN)make clean$(NC)        - Remove binários gerados"
	@echo "  $(GREEN)make docker-build$(NC) - Constrói a imagem Docker"
	@echo "  $(GREEN)make docker-up$(NC)    - Inicia os serviços com Docker Compose"
	@echo "  $(GREEN)make docker-down$(NC)  - Para os serviços do Docker Compose"
