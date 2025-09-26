package service

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	llmApi "github.com/dhruv304c2/ai-guru-be.git/service/llm_api"
	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

// Start launches an HTTP server that exposes a simple hello world endpoint.
// The server listens on the provided address until ListenAndServe returns.
func Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Hello, world!"})
	})

	_ = godotenv.Load()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Fatal(err)
	}

	//Handlers
	mux.HandleFunc("/llm/chat", llmApi.ChatHandler(client))
	mux.HandleFunc("/llm/chat/partial", llmApi.ChatStreamHandler(client))

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("HTTP service listening on %s", addr)
	return server.ListenAndServe()
}
