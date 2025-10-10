package main

import (
	"fmt"
	"log"
	"time"
)

func funcaoComLogCompleto(nome string) (err error) {
	inicio := time.Now()

	// Defer com log completo de execução
	defer func() {
		duracao := time.Since(inicio)

		// Recovery de panic
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recuperado: %v", r)
		}

		// Log baseado no resultado
		if err != nil {
			log.Printf("ERRO: função '%s' falhou após %v - erro: %v", nome, duracao, err)
		} else {
			log.Printf("SUCESSO: função '%s' executada com sucesso em %v", nome, duracao)
		}
	}()

	// Simular processamento
	time.Sleep(100 * time.Millisecond)

	// Diferentes cenários para teste
	switch nome {
	case "sucesso":
		fmt.Println("Processamento realizado com sucesso")
		return nil
	case "erro":
		return fmt.Errorf("erro simulado para teste")
	case "panic":
		panic("panic simulado para teste")
	default:
		return nil
	}
}

func main() {
	funcaoComLogCompleto("erro")
}
