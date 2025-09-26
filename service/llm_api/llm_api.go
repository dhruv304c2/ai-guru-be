package llmApi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

const generatePromptSeed = `Based on the conversation so far and the last user message, generate a list of short possible user prompts/questions the user might want to ask next.
- Keep each prompt under 10 words.
- Respond with a raw JSON array of strings, for example: ["Question 1", "Question 2"].
- The first character of your response must be '[' and the last must be ']'.
- Output must be valid JSON without backticks, code fences, markdown, or commentary.
- If no prompts apply, respond with [].`

// ---------------------------
// Shared helpers (single parser + content builder)
// ---------------------------

func decodeChatRequest(w http.ResponseWriter, r *http.Request) (ChatRequest, error) {
	var req ChatRequest

	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		return req, errors.New("method not allowed")
	}

	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if ct != "application/json" {
		return req, errors.New("Content-Type must be application/json")
	}

	// Limit payload to 1 MiB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, errors.New("invalid JSON: " + err.Error())
	}
	if strings.TrimSpace(req.Message) == "" && len(req.History) == 0 {
		return req, errors.New("message required (or history/seed)")
	}
	return req, nil
}

func buildContents(req ChatRequest) (contents []*genai.Content, model string) {
	contents = append(contents, &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{{Text: defaultSeed}},
	})

	for _, m := range req.History {
		if s := strings.TrimSpace(m.Content); s != "" {
			contents = append(contents, &genai.Content{
				Role:  toGenAIRole(m.Role),
				Parts: []*genai.Part{{Text: s}},
			})
		}
	}
	if msg := strings.TrimSpace(req.Message); msg != "" {
		contents = append(contents, &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{{Text: msg}},
		})
	}

	model = req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return contents, model
}

func buildContentsForPrompt(req ChatRequest) (contents []*genai.Content, model string) {
	contents = append(contents, &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{{Text: defaultSeed}},
	})

	for _, m := range req.History {
		if s := strings.TrimSpace(m.Content); s != "" {
			contents = append(contents, &genai.Content{
				Role:  toGenAIRole(m.Role),
				Parts: []*genai.Part{{Text: s}},
			})
		}
	}

	contents = append(contents, &genai.Content{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{{Text: generatePromptSeed}},
	})

	if msg := strings.TrimSpace(req.Message); msg != "" {
		contents = append(contents, &genai.Content{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{{Text: msg}},
		})
	}

	model = req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return contents, model
}

// ---------------------------
// Existing non-stream handler
// ---------------------------

func ChatHandler(client *genai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeChatRequest(w, r)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case err.Error() == "method not allowed":
				status = http.StatusMethodNotAllowed
			case strings.Contains(err.Error(), "Content-Type"):
				status = http.StatusUnsupportedMediaType
			}
			WriteJSON(w, status, JsonErr{err.Error()})
			return
		}

		content, model := buildContents(req)

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
			// Never return a blank body
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

func GenerateUserPromptHandler(client *genai.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeChatRequest(w, r)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case err.Error() == "method not allowed":
				status = http.StatusMethodNotAllowed
			case strings.Contains(err.Error(), "Content-Type"):
				status = http.StatusUnsupportedMediaType
			}
			WriteJSON(w, status, JsonErr{err.Error()})
			return
		}

		content, model := buildContentsForPrompt(req)

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

		raw := strings.TrimSpace(res.Text())

		var prompts []string
		if raw != "" {
			if err := json.Unmarshal([]byte(raw), &prompts); err != nil {
				log.Printf("prompt handler: unable to parse response as JSON array: %v", err)
			}
		}
		if prompts == nil {
			prompts = []string{}
		}

		WriteJSON(w, http.StatusOK, PromptResponse{
			Prompts: prompts,
			Model:   model,
		})
	}
}

// ---------------------------
// Streaming handler (SSE)
// ---------------------------
func ChatStreamHandler(client *genai.Client) http.HandlerFunc {
	type flusher interface{ Flush() }

	writeSSE := func(w http.ResponseWriter, event, data string) error {
		if event != "" {
			if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
				return err
			}
		}
		for _, line := range strings.Split(data, "\n") {
			if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, "\n"); err != nil {
			return err
		}
		if f, ok := w.(flusher); ok {
			f.Flush()
		}
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeChatRequest(w, r)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case err.Error() == "method not allowed":
				status = http.StatusMethodNotAllowed
			case strings.Contains(err.Error(), "Content-Type"):
				status = http.StatusUnsupportedMediaType
			}
			WriteJSON(w, status, JsonErr{err.Error()})
			return
		}

		contents, model := buildContents(req)

		// SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache, no-transform")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no") // helps with nginx

		if _, ok := w.(flusher); !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		_ = writeSSE(w, "start", fmt.Sprintf(`{"model":%q}`, model))

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()

		it := client.Models.GenerateContentStream(ctx, model, contents, nil)

		var full strings.Builder

		for resp, err := range it {
			if err != nil {
				_ = writeSSE(w, "error", fmt.Sprintf(`{"error":%q}`, "model error: "+err.Error()))
				_ = writeSSE(w, "done", "{}")
				return
			}
			if resp == nil {
				continue
			}
			for _, cand := range resp.Candidates {
				if cand == nil || cand.Content == nil {
					continue
				}
				for _, part := range cand.Content.Parts {
					if part == nil || part.Text == "" {
						continue
					}
					full.WriteString(part.Text)
					if err := writeSSE(w, "partial", fmt.Sprintf(`{"part":%q}`, part.Text)); err != nil {
						log.Printf("llm chat stream write error: %v", err) // client likely disconnected
						return
					}
				}
			}
		}

		_ = writeSSE(w, "complete", fmt.Sprintf(`{"reply":%q,"model":%q}`, full.String(), model))
		_ = writeSSE(w, "done", "{}")
	}
}

// Your existing role mapper (kept as-is)
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
