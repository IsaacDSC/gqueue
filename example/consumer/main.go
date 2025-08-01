package main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		log.Println("[*] Received request:", r.Method, r.URL.Path)

		defer r.Body.Close()
		b, _ := io.ReadAll(r.Body) // Read the body to avoid closing it prematurely

		log.Println(string(b))

		w.Write([]byte("Hello, Webhook!"))
	})

	log.Println("[*] Server started on :8081")
	http.ListenAndServe(":8081", nil)
}

// curl example:
// curl -X POST http://localhost:8081/notifications/new-user -d '{"message": "Hello, Webhook!"}' -H "Content-Type: application/json"
