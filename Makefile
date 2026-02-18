# Makefile para o projeto webhook

# Variáveis
GO=go
APP_NAME=webhook
PORT=8080
MOCKGEN=$(GO) run go.uber.org/mock/mockgen@latest

# Cores para output
GREEN=\033[0;32m
BLUE=\033[0;34m
YELLOW=\033[0;33m
NC=\033[0m # No Color

# Comandos principais
.PHONY: all build run test clean load-test run-worker run-webhook run-all generate-mocks update-mocks install-mockgen check-mocks test-with-mocks clean-mocks lint security coverage check-coverage coverage-check ci

# Comandos por padrão
all: help

# Executar todos os checks do CI localmente
ci: lint security test build clean
	@echo "$(GREEN)✅ Todos os checks do CI passaram com sucesso!$(NC)"

# Construir o aplicativo
build:
	@echo "$(GREEN)Construindo aplicação...$(NC)"
	@$(GO) build -o $(APP_NAME) ./cmd/api/main.go

# Rodar teste de carga
load-test:
	@echo "$(YELLOW)Executando teste de carga...$(NC)"
	@$(GO) run ./cmd/loadtest/loadtest.go

# Iniciar worker
run-worker:
	@echo "$(BLUE)Iniciando serviço worker...$(NC)"
	@$(GO) run ./cmd/api/main.go --service=worker

# Iniciar webhook (API)
run-webhook:
	@echo "$(BLUE)Iniciando serviço webhook (API)...$(NC)"
	@$(GO) run ./cmd/api/main.go --service=webhook

# Iniciar ambos serviços
run-all:
	@echo "$(BLUE)Iniciando todos os serviços (worker e webhook)...$(NC)"
	@$(GO) run ./cmd/api/main.go --service=all

# Limpar binários gerados
clean:
	@echo "$(GREEN)Limpando binários...$(NC)"
	@rm -f $(APP_NAME)

# Executar testes
test:
	@echo "$(GREEN)Executando testes...$(NC)"
	GO_ENV=test $(GO) test ./... -v

# Executar testes com cobertura (excluindo pastas específicas)
coverage:
	@echo "$(GREEN)Executando testes com cobertura...$(NC)"
	@echo "$(BLUE)Excluindo: example/, cmd/, docs/, deployment/, *_mock.go$(NC)"
	@GO_ENV=test $(GO) test $$(go list ./... | grep -v '/example/' | grep -v '/cmd/' | grep -v '/docs/' | grep -v '/deployment/') -coverprofile=coverage.out -covermode=atomic
	@# Remove mock files from coverage report
	@grep -v '_mock.go' coverage.out > coverage_filtered.out || true
	@mv coverage_filtered.out coverage.out || true
	@$(GO) tool cover -func=coverage.out
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(BLUE)Relatório HTML gerado em coverage.html$(NC)"

# Verificar se cobertura atende 80% mínimo (excluindo pastas específicas)
check-coverage:
	@echo "$(GREEN)Verificando cobertura mínima de 80%...$(NC)"
	@echo "$(BLUE)Excluindo: example/, cmd/, docs/, deployment/, *_mock.go$(NC)"
	@GO_ENV=test $(GO) test $$(go list ./... | grep -v '/example/' | grep -v '/cmd/' | grep -v '/docs/' | grep -v '/deployment/') -coverprofile=coverage.out -covermode=atomic
	@# Remove mock files from coverage report
	@grep -v '_mock.go' coverage.out > coverage_filtered.out || true
	@mv coverage_filtered.out coverage.out || true
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "$(BLUE)Cobertura total: $${COVERAGE}%$(NC)"; \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "$(YELLOW)❌ FALHA: Cobertura ($${COVERAGE}%) abaixo do mínimo de 80%$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)✅ SUCESSO: Cobertura atende ao mínimo de 80%$(NC)"; \
	fi

# Executar lint
lint:
	@echo "$(GREEN)Executando lint...$(NC)"
	@echo "$(BLUE)Verificando formatação...$(NC)"
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "$(YELLOW)⚠️  Os seguintes arquivos não estão formatados corretamente:$(NC)"; \
		echo "$$unformatted"; \
		echo "$(BLUE)Execute 'gofmt -w .' para corrigir$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Executando go vet...$(NC)"
	@$(GO) vet ./...
	@echo "$(BLUE)Verificando go mod tidy...$(NC)"
	@$(GO) mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "$(YELLOW)⚠️  go.mod ou go.sum não estão atualizados$(NC)"; \
		echo "$(BLUE)Execute 'go mod tidy' para corrigir$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)✅ Lint passou com sucesso!$(NC)"

# Executar scan de segurança com govulncheck
security:
	@echo "$(GREEN)Executando scan de segurança com govulncheck...$(NC)"
	@$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "$(GREEN)Executando scan de segurança com gosec...$(NC)"
	@$(GO) run github.com/securego/gosec/v2/cmd/gosec@latest -exclude-generated -severity=high -confidence=high ./...
	@echo "$(GREEN)✅ Security scan passou com sucesso!$(NC)"

# Verificar cobertura dos arquivos commitados (simples)
coverage-check:
	@echo "$(GREEN)🔍 Verificando cobertura dos arquivos commitados$(NC)"
	@# Detectar branch principal
	@MAIN_BRANCH=$$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main"); \
	if ! git rev-parse --verify origin/$$MAIN_BRANCH >/dev/null 2>&1; then \
		MAIN_BRANCH="master"; \
	fi; \
	echo "$(BLUE)Comparando com: $$MAIN_BRANCH$(NC)"; \
	CHANGED_FILES=$$(git diff --name-only origin/$$MAIN_BRANCH...HEAD | grep '\.go$$' | grep -v '_test\.go$$' | grep -v '_mock\.go$$' | grep -v '^example/' | grep -v '^cmd/' | grep -v '^docs/' | grep -v '^deployment/' | xargs -I {} sh -c 'test -f "{}" && echo "{}"' || true); \
	if [ -z "$$CHANGED_FILES" ]; then \
		echo "$(YELLOW)❌ Nenhum arquivo Go commitado encontrado$(NC)"; \
		exit 0; \
	fi; \
	echo "$(BLUE)Arquivos commitados:$(NC)"; \
	echo "$$CHANGED_FILES" | sed 's/^/  - /'; \
	echo ""; \
	echo "$(BLUE)Executando testes...$(NC)"; \
	GO_ENV=test $(GO) test $$(go list ./... | grep -v '/example/' | grep -v '/cmd/' | grep -v '/docs/' | grep -v '/deployment/') -coverprofile=coverage.out -covermode=atomic >/dev/null 2>&1; \
	grep -v '_mock.go' coverage.out > coverage_filtered.out || true; \
	mv coverage_filtered.out coverage.out || true; \
	echo ""; \
	echo "$(BLUE)📊 COBERTURA:$(NC)"; \
	FAILED_COUNT=0; \
	for file in $$CHANGED_FILES; do \
		COVERAGE_LINE=$$($(GO) tool cover -func=coverage.out | grep "$$file" | head -1); \
		if [ -n "$$COVERAGE_LINE" ]; then \
			COVERAGE_PCT=$$(echo "$$COVERAGE_LINE" | awk '{print $$3}' | sed 's/%//'); \
			if [ $$(echo "$$COVERAGE_PCT < 80" | bc -l 2>/dev/null || echo 0) -eq 1 ]; then \
				echo "  $(YELLOW)❌ $$file: $${COVERAGE_PCT}%$(NC)"; \
				FAILED_COUNT=$$((FAILED_COUNT + 1)); \
			else \
				echo "  $(GREEN)✅ $$file: $${COVERAGE_PCT}%$(NC)"; \
			fi; \
		else \
			echo "  $(YELLOW)❌ $$file: 0.0% (sem testes)$(NC)"; \
			FAILED_COUNT=$$((FAILED_COUNT + 1)); \
		fi; \
	done; \
	echo ""; \
	if [ $$FAILED_COUNT -gt 0 ]; then \
		echo "$(YELLOW)⚠️  $$FAILED_COUNT arquivo(s) precisam de mais testes$(NC)"; \
		echo "$(BLUE)Dica: Execute 'make coverage' e abra coverage.html para ver as linhas específicas$(NC)"; \
	else \
		echo "$(GREEN)✅ Todos os arquivos atendem ao critério de 80%$(NC)"; \
	fi

# Executar testes com verificação de mocks
test-with-mocks: check-mocks test
	@echo "$(GREEN)Testes executados com mocks verificados!$(NC)"

# Executar testes do fetcher
test-fetcher:
	@echo "$(GREEN)Executando testes do fetcher...$(NC)"
	@WQ_QUEUES='{"internal.default":1,"external.default":1}' $(GO) test ./internal/fetcher -v

# Executar testes do deadletter
test-deadletter:
	@echo "$(GREEN)Executando testes do deadletter...$(NC)"
	@GO_ENV=test WQ_QUEUES='{"internal.default":1,"external.default":1}' $(GO) test ./internal/wtrhandler -v -run "TestNewDeadLatterQueue|TestDeadLetter"

# Executar testes do insights
test-insights:
	@echo "$(GREEN)Executando testes do insights...$(NC)"
	@GO_ENV=test $(GO) test ./internal/storests -v

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

# Gerar mocks
install-mockgen:
	@echo "$(GREEN)Instalando mockgen...$(NC)"
	@$(GO) install go.uber.org/mock/mockgen@latest

generate-mocks: install-mockgen clean-mocks
	@echo "$(GREEN)Gerando mocks...$(NC)"
	@echo "$(BLUE)Gerando mock para Repository...$(NC)"
	@echo "$(BLUE)Gerando mock para DeadLetter...$(NC)"
	@$(MOCKGEN) -source=internal/wtrhandler/deadletter_asynq_handle.go -destination=internal/wtrhandler/deadletter_mock.go -package=wtrhandler DeadLetterStore
	@echo "$(BLUE)Gerando mock para Fetcher...$(NC)"
	@$(MOCKGEN) -source=internal/wtrhandler/request_handle_asynq.go -destination=internal/wtrhandler/fetcher_mock.go -package=wtrhandler Fetcher
	@echo "$(BLUE)Gerando mock para Cache...$(NC)"
	@$(MOCKGEN) -source=pkg/cachemanager/adapter.go -destination=pkg/cachemanager/cache_mock.go -package=cachemanager
	@echo "$(BLUE)Gerando mock para Backoffice Repository...$(NC)"
	@$(MOCKGEN) -source=internal/backoffice/interfaces.go -destination=internal/backoffice/repository_mock.go -package=backoffice
	@echo "$(BLUE)Gerando mock para Publisher...$(NC)"
	@$(MOCKGEN) -source=pkg/pubadapter/adapter.go -destination=pkg/publisher/publisher_task_mock.go -package=publisher
	@echo "$(BLUE)Gerando mock para Publisher em pubadapter...$(NC)"
	@$(MOCKGEN) -source=pkg/pubadapter/adapter.go -destination=pkg/pubadapter/publisher_task_mock.go -package=pubadapter
	@echo "$(BLUE)Gerando mock para PublisherInsights...$(NC)"
	# TODO: está sendo criado junto com outro por estar no mesmo arquivo, separar interfaces em arquivos diferentes
	# @$(MOCKGEN) -source=tmp/publisher_insights.go -destination=internal/wtrhandler/mock_publisher_insights.go -package=wtrhandler
	@echo "$(BLUE)Gerando mock para ConsumerInsights...$(NC)"
	# TODO: está sendo criado junto com outro por estar no mesmo arquivo, separar interfaces em arquivos diferentes
	# @$(MOCKGEN) -source=tmp/consumer_insights.go -destination=internal/wtrhandler/mock_consumer_insights.go -package=wtrhandler
	@echo "$(GREEN)Mocks gerados com sucesso!$(NC)"

update-mocks: generate-mocks
	@echo "$(GREEN)Mocks atualizados!$(NC)"

check-mocks:
	@echo "$(GREEN)Verificando se os mocks existem...$(NC)"
	@if [ ! -f "internal/wtrhandler/repository_mock.go" ]; then \
		echo "$(YELLOW)⚠️  Repository mock não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	@if [ ! -f "internal/wtrhandler/deadletter_mock.go" ]; then \
		echo "$(YELLOW)⚠️  DeadLetter mock não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	@if [ ! -f "internal/wtrhandler/fetcher_mock.go" ]; then \
		echo "$(YELLOW)⚠️  Fetcher mock não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	@if [ ! -f "pkg/cachemanager/cache_mock.go" ]; then \
		echo "$(YELLOW)⚠️  Cache mock não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	@if [ ! -f "pkg/publisher/publisher_task_mock.go" ]; then \
		echo "$(YELLOW)⚠️  Publisher mock não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	@if [ ! -f "pkg/pubadapter/publisher_task_mock.go" ]; then \
		echo "$(YELLOW)⚠️  Publisher mock em pubadapter não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	@if [ ! -f "internal/backoffice/repository_mock.go" ]; then \
		echo "$(YELLOW)⚠️  Backoffice Repository mock não encontrado!$(NC)"; \
		echo "Execute 'make generate-mocks' para gerar os mocks"; \
		exit 1; \
	fi
	# TODO: Descomentar quando os mocks de insights forem implementados
	# @if [ ! -f "internal/wtrhandler/mock_publisher_insights.go" ]; then \
	#	echo "$(YELLOW)⚠️  PublisherInsights mock não encontrado!$(NC)"; \
	#	echo "Execute 'make generate-mocks' para gerar os mocks"; \
	#	exit 1; \
	# fi
	# @if [ ! -f "internal/wtrhandler/mock_consumer_insights.go" ]; then \
	#	echo "$(YELLOW)⚠️  ConsumerInsights mock não encontrado!$(NC)"; \
	#	echo "Execute 'make generate-mocks' para gerar os mocks"; \
	#	exit 1; \
	# fi
	@echo "$(GREEN)✅ Todos os mocks existem!$(NC)"
	@echo "$(BLUE)💡 Para regenerar todos os mocks, execute: make update-mocks$(NC)"

clean-mocks:
	@echo "$(GREEN)Removendo mocks...$(NC)"
	@echo "$(BLUE)Removendo Repository mock...$(NC)"
	@rm -f internal/wtrhandler/repository_mock.go
	@echo "$(BLUE)Removendo DeadLetter mock...$(NC)"
	@rm -f internal/wtrhandler/deadletter_mock.go
	@echo "$(BLUE)Removendo Fetcher mock...$(NC)"
	@rm -f internal/wtrhandler/fetcher_mock.go
	@echo "$(BLUE)Removendo Cache mock...$(NC)"
	@rm -f pkg/cachemanager/cache_mock.go
	@echo "$(BLUE)Removendo Publisher mock...$(NC)"
	@rm -f pkg/publisher/publisher_task_mock.go
	@echo "$(BLUE)Removendo Publisher mock em pubadapter...$(NC)"
	@rm -f pkg/pubadapter/publisher_task_mock.go
	@echo "$(BLUE)Removendo Backoffice Repository mock...$(NC)"
	@rm -f internal/backoffice/repository_mock.go
	@echo "$(BLUE)Removendo PublisherInsights mock...$(NC)"
	@rm -f internal/wtrhandler/mock_publisher_insights.go
	@echo "$(BLUE)Removendo ConsumerInsights mock...$(NC)"
	@rm -f internal/wtrhandler/mock_consumer_insights.go
	@echo "$(GREEN)Mocks removidos com sucesso!$(NC)"
	@echo "$(BLUE)💡 Para gerar novos mocks, execute: make generate-mocks$(NC)"

# Ajuda
help:
	@echo "$(YELLOW)Comandos disponíveis:$(NC)"
	@echo "  $(GREEN)make ci$(NC)              - Executa todos os checks do CI (lint, security, test, build)"
	@echo "  $(GREEN)make build$(NC)           - Constrói a aplicação"
	@echo "  $(GREEN)make run-worker$(NC)      - Executa o serviço worker"
	@echo "  $(GREEN)make run-webhook$(NC)     - Executa o serviço webhook (API)"
	@echo "  $(GREEN)make run-all$(NC)         - Executa ambos os serviços"
	@echo "  $(GREEN)make load-test$(NC)       - Executa teste de carga"
	@echo "  $(GREEN)make test$(NC)            - Executa os testes"
	@echo "  $(GREEN)make coverage$(NC)        - Executa testes com relatório de cobertura (exclui: example/, cmd/, docs/, deployment/, *_mock.go)"
	@echo "  $(GREEN)make check-coverage$(NC)  - Verifica se cobertura >= 80% (exclui: example/, cmd/, docs/, deployment/, *_mock.go)"
	@echo "  $(GREEN)make coverage-check$(NC)  - 🔍 Verifica cobertura dos arquivos commitados (SIMPLES)"
	@echo "  $(GREEN)make lint$(NC)            - Executa lint (fmt, vet, mod tidy)"
	@echo "  $(GREEN)make security$(NC)        - Executa scan de segurança com govulncheck"
	@echo "  $(GREEN)make test-with-mocks$(NC) - Executa os testes com verificação de mocks"
	@echo "  $(GREEN)make test-fetcher$(NC)    - Executa os testes do fetcher"
	@echo "  $(GREEN)make test-deadletter$(NC) - Executa os testes do deadletter"
	@echo "  $(GREEN)make test-insights$(NC)   - Executa os testes do insights"
	@echo "  $(GREEN)make generate-mocks$(NC)  - Gera todos os mocks (Repository, DeadLetter, Fetcher, Cache, Publisher, Backoffice Repository)"
	@echo "  $(GREEN)make update-mocks$(NC)    - Atualiza todos os mocks"
	@echo "  $(GREEN)make check-mocks$(NC)     - Verifica se os mocks existem"
	@echo "  $(GREEN)make clean-mocks$(NC)     - Remove todos os mocks"
	@echo "  $(GREEN)make install-mockgen$(NC) - Instala a ferramenta mockgen"
	@echo "  $(GREEN)make clean$(NC)           - Remove binários gerados"
	@echo "  $(GREEN)make docker-build$(NC)    - Constrói a imagem Docker"
	@echo "  $(GREEN)make docker-up$(NC)       - Inicia os serviços com Docker Compose"
	@echo "  $(GREEN)make docker-down$(NC)     - Para os serviços do Docker Compose"
	@echo ""
	@echo "$(YELLOW)Mocks gerados:$(NC)"
	@echo "  $(BLUE)• Repository$(NC)          - internal/wtrhandler/repository_mock.go"
	@echo "  $(BLUE)• DeadLetter$(NC)          - internal/wtrhandler/deadletter_mock.go"
	@echo "  $(BLUE)• Fetcher$(NC)             - internal/wtrhandler/fetcher_mock.go"
	@echo "  $(BLUE)• Cache$(NC)               - pkg/cachemanager/cache_mock.go"
	@echo "  $(BLUE)• Publisher$(NC)           - pkg/publisher/publisher_task_mock.go"
	@echo "  $(BLUE)• Publisher (pubadapter)$(NC) - pkg/pubadapter/publisher_task_mock.go"
	@echo "  $(BLUE)• Backoffice Repository$(NC) - internal/backoffice/repository_mock.go"
