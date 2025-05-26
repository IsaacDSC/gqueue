package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func main() {
	// Inicializar gerador de dados aleatórios
	gofakeit.Seed(time.Now().UnixNano())

	// Configurações do teste de carga
	rate := vegeta.Rate{Freq: 50, Per: time.Second} // 50 requisições por segundo
	duration := 30 * time.Second                    // duração do teste

	// Criar atacante
	attacker := vegeta.NewAttacker()

	// Executar teste
	var metrics vegeta.Metrics
	for res := range attacker.Attack(createTargeter(), rate, duration, "Load Test") {
		metrics.Add(res)
	}
	metrics.Close()

	// Mostrar resultados
	fmt.Printf("99th percentile: %s\n", metrics.Latencies.P99)
	fmt.Printf("95th percentile: %s\n", metrics.Latencies.P95)
	fmt.Printf("Mean: %s\n", metrics.Latencies.Mean)
	fmt.Printf("Max: %s\n", metrics.Latencies.Max)
	fmt.Printf("Requests per second: %.2f\n", metrics.Rate)
	fmt.Printf("Success ratio: %.2f%%\n", metrics.Success*100)
	fmt.Printf("Status codes: %v\n", metrics.StatusCodes)
	fmt.Printf("Total requests: %d\n", metrics.Requests)

	// Salvar relatório detalhado em formato texto
	fmt.Println("\n=== Relatório Detalhado ===")
	reporter := vegeta.NewTextReporter(&metrics)
	reporter.Report(os.Stdout)
}

// createTargeter cria uma função targeter que gera usuários aleatórios para cada requisição
func createTargeter() vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		// Gerar dados falsos para o usuário
		userId := uuid.New().String()
		name := gofakeit.Name()
		email := gofakeit.Email()

		// Preparar payload
		payload := fmt.Sprintf(`{
  "event_name": "user.created",
  "data": {
    "userId": "%s",
    "email": "%s",
    "name": "%s"
  }
}`, userId, email, name)

		// Configurar o target
		tgt.Method = "POST"
		tgt.URL = "http://localhost:8080/event/publisher"
		tgt.Body = []byte(payload)
		tgt.Header = http.Header{
			"Content-Type": {"application/json"},
		}

		return nil
	}
}
