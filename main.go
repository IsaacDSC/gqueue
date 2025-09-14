package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/notify", func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		fmt.Println("Received payload:", payload)
		w.WriteHeader(http.StatusOK)
	})

	log.Print("Starting server on :8888\n")
	if err := http.ListenAndServe(":8888", nil); err != nil {
		panic(err)
	}
}
