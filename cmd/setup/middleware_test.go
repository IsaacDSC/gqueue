package setup

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddleware_DefaultConfig(t *testing.T) {
	// Handler simples para teste
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Aplicar middleware de CORS
	corsHandler := CORSMiddleware(handler)

	// Criar requisição de teste
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	// Criar ResponseRecorder
	rr := httptest.NewRecorder()

	// Executar requisição
	corsHandler.ServeHTTP(rr, req)

	// Verificar headers de CORS
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "*")
	}

	if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods = %v, want %v", got, "GET, POST, PUT, DELETE, OPTIONS")
	}

	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization, X-Requested-With" {
		t.Errorf("Access-Control-Allow-Headers = %v, want %v", got, "Content-Type, Authorization, X-Requested-With")
	}

	if got := rr.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("Access-Control-Max-Age = %v, want %v", got, "86400")
	}

	// Verificar que a resposta chegou ao handler
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", rr.Code, http.StatusOK)
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	// Handler simples para teste
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Aplicar middleware de CORS
	corsHandler := CORSMiddleware(handler)

	// Criar requisição OPTIONS (preflight)
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	// Criar ResponseRecorder
	rr := httptest.NewRecorder()

	// Executar requisição
	corsHandler.ServeHTTP(rr, req)

	// Verificar que retorna 204 No Content para OPTIONS
	if rr.Code != http.StatusNoContent {
		t.Errorf("Status code for OPTIONS = %v, want %v", rr.Code, http.StatusNoContent)
	}

	// Verificar que o handler original não foi chamado (body vazio)
	if rr.Body.String() != "" {
		t.Errorf("Body should be empty for OPTIONS request, got %v", rr.Body.String())
	}

	// Verificar headers de CORS
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "*")
	}
}

func TestCORSMiddlewareWithConfig_CustomOrigins(t *testing.T) {
	// Configuração personalizada
	config := CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "https://example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           3600,
	}

	// Handler simples para teste
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Aplicar middleware de CORS com configuração personalizada
	corsHandler := CORSMiddlewareWithConfig(config)(handler)

	// Teste com origem permitida
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %v, want %v", got, "http://localhost:3000")
	}

	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %v, want %v", got, "true")
	}

	if got := rr.Header().Get("Access-Control-Max-Age"); got != "3600" {
		t.Errorf("Access-Control-Max-Age = %v, want %v", got, "3600")
	}
}

func TestCORSMiddlewareWithConfig_DisallowedOrigin(t *testing.T) {
	// Configuração personalizada com origens específicas
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	}

	// Handler simples para teste
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Aplicar middleware de CORS com configuração personalizada
	corsHandler := CORSMiddlewareWithConfig(config)(handler)

	// Teste com origem não permitida
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://malicious-site.com")

	rr := httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	// Access-Control-Allow-Origin não deve ser definido para origem não permitida
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin should not be set for disallowed origin, got %v", got)
	}

	// O handler ainda deve ser executado
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", rr.Code, http.StatusOK)
	}
}

func TestCORSMiddlewareWithConfig_ExposedHeaders(t *testing.T) {
	// Configuração com headers expostos
	config := CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET"},
		AllowedHeaders: []string{"Content-Type"},
		ExposedHeaders: []string{"X-Total-Count", "X-Page-Number"},
	}

	// Handler simples para teste
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Total-Count", "100")
		w.Header().Set("X-Page-Number", "1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Aplicar middleware de CORS com configuração personalizada
	corsHandler := CORSMiddlewareWithConfig(config)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	rr := httptest.NewRecorder()
	corsHandler.ServeHTTP(rr, req)

	expectedExposed := "X-Total-Count, X-Page-Number"
	if got := rr.Header().Get("Access-Control-Expose-Headers"); got != expectedExposed {
		t.Errorf("Access-Control-Expose-Headers = %v, want %v", got, expectedExposed)
	}

	// Verificar que os headers customizados foram definidos
	if got := rr.Header().Get("X-Total-Count"); got != "100" {
		t.Errorf("X-Total-Count = %v, want %v", got, "100")
	}
}

func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	// Verificar valores padrão
	if len(config.AllowedOrigins) != 1 || config.AllowedOrigins[0] != "*" {
		t.Errorf("Default AllowedOrigins = %v, want %v", config.AllowedOrigins, []string{"*"})
	}

	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	if len(config.AllowedMethods) != len(expectedMethods) {
		t.Errorf("Default AllowedMethods length = %v, want %v", len(config.AllowedMethods), len(expectedMethods))
	}

	expectedHeaders := []string{"Content-Type", "Authorization", "X-Requested-With"}
	if len(config.AllowedHeaders) != len(expectedHeaders) {
		t.Errorf("Default AllowedHeaders length = %v, want %v", len(config.AllowedHeaders), len(expectedHeaders))
	}

	if config.AllowCredentials != false {
		t.Errorf("Default AllowCredentials = %v, want %v", config.AllowCredentials, false)
	}

	if config.MaxAge != 86400 {
		t.Errorf("Default MaxAge = %v, want %v", config.MaxAge, 86400)
	}
}
