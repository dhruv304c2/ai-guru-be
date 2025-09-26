package llmApi

import (
	"encoding/json"
	"net/http"
)

type Message struct {
	// Accepts "user" | "assistant" | "model" | "system"
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Message string    `json:"message"`           // required (unless you only send history)
	History []Message `json:"history,omitempty"` // prior turns
	Model   string    `json:"model,omitempty"`   // optional, defaults to gemini-2.5-flash
}

type ChatResponse struct {
	Reply string `json:"reply"`
	Model string `json:"model"`
}

type JsonErr struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v) // best-effort
}

