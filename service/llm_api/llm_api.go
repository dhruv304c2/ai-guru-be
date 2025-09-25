package llmApi

import (
	"context"
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

const defaultSeed = `You are an enlightened Guru who has mastered the ancient wisdom of Hindu scriptures—both Śruti (Vedas, Brāhmaṇas, Āraṇyakas, Upaniṣads) including ayurveda and Smṛti (Itihāsas like Rāmāyaṇa & Mahābhārata, Purāṇas, Dharmaśāstras, Āgamas, Tantras, Sūtras & Śāstras such as Yoga, Vedānta, Nyāya, Sāṃkhya, etc.).
Your role is to serve seekers in the modern world by making this timeless wisdom clear, relatable, and practical.
Guidelines:
1) Assess first, 2) Adapt teaching, 3) Śāstrārtha, 4) Bridge old & new, 5) Compassionate, crisp, authoritative tone, 6) Mission: clarity & transformation, 7) Keep responses crisp.`

func toGenAIRole(s string) string {
	switch strings.ToLower(s) {
	case "assistant", "model", "ai":
		return genai.RoleModel
	case "system":
		return genai.RoleUser
	default:
		return genai.RoleUser
	}
}

func ChatHandler(client *genai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			WriteJSON(w, http.StatusMethodNotAllowed, JsonErr{"method not allowed"})
			return
		}

		// Accept application/json or application/json; charset=utf-8
		ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if ct != "application/json" {
			WriteJSON(w, http.StatusUnsupportedMediaType, JsonErr{"Content-Type must be application/json"})
			return
		}

		// Limit payload to 1 MiB
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()

		var req ChatRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			WriteJSON(w, http.StatusBadRequest, JsonErr{"invalid JSON: " + err.Error()})
			return
		}
		if strings.TrimSpace(req.Message) == "" && len(req.History) == 0 && strings.TrimSpace(req.Seed) == "" {
			WriteJSON(w, http.StatusBadRequest, JsonErr{"message required (or history/seed)"})
			return
		}

		// Build conversation
		var content []*genai.Content

		seed := strings.TrimSpace(req.Seed)
		if seed == "" && len(req.History) == 0 {
			seed = defaultSeed
		}
		if seed != "" {
			content = append(content, &genai.Content{
				Role:  genai.RoleUser, // seed as user instruction
				Parts: []*genai.Part{{Text: seed}},
			})
		}

		for _, m := range req.History {
			if s := strings.TrimSpace(m.Content); s != "" {
				content = append(content, &genai.Content{
					Role:  toGenAIRole(m.Role),
					Parts: []*genai.Part{{Text: s}},
				})
			}
		}

		if msg := strings.TrimSpace(req.Message); msg != "" {
			content = append(content, &genai.Content{
				Role:  genai.RoleUser,
				Parts: []*genai.Part{{Text: msg}},
			})
		}

		model := req.Model
		if model == "" {
			model = "gemini-2.5-flash"
		}

		// Tie to request and bound time
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		res, err := client.Models.GenerateContent(ctx, model, content, nil)
		if err != nil {
			WriteJSON(w, http.StatusBadGateway, JsonErr{"model error: " + err.Error()})
			return
		}
		if res == nil {
			log.Printf("llm chat: nil response from model %q", model)
			WriteJSON(w, http.StatusBadGateway, JsonErr{"model error: empty response"})
			return
		}

		reply := strings.TrimSpace(res.Text())
		if reply == "" {
			// Never return a blank body: surface an explicit note
			WriteJSON(w, http.StatusOK, struct {
				Reply string `json:"reply"`
				Model string `json:"model"`
				Note  string `json:"note"`
				TS    string `json:"ts"`
			}{
				Reply: "",
				Model: model,
				Note:  "empty model reply (blocked or no text parts)",
				TS:    time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		WriteJSON(w, http.StatusOK, ChatResponse{
			Reply: reply,
			Model: model,
		})
	}
}
