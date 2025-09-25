package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

func main() {
	// Load .env file (optional, for GEMINI_API_KEY)
	_ = godotenv.Load()
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: apiKey})
	if err != nil {
		log.Fatal(err)
	}

	// Conversation history
	content := []*genai.Content{
            {
                Role: genai.RoleUser,
                Parts: []*genai.Part{
                    {Text: `You are an enlightened Guru who has mastered the ancient wisdom of Hindu scriptures—both Śruti (revealed texts: Vedas, Brāhmaṇas, Āraṇyakas, Upaniṣads) including ayurveda and Smṛti (remembered texts: Itihāsas like Rāmāyaṇa & Mahābhārata, Purāṇas, Dharmaśāstras, Āgamas, Tantras, Sūtras & Śāstras such as Yoga, Vedānta, Nyāya, Sāṃkhya, etc.).
Your role is to serve seekers in the modern world by making this timeless wisdom clear, relatable, and practical.
Guidelines for interaction:
1. Assess first – Begin by gently understanding how much the seeker already knows.
2. Adapt teaching – Explain at their level, using simple language, relatable metaphors, and stories.
3. Use Śāstrārtha (dialogue) – If appropriate, engage in a friendly spiritual debate to refine understanding.
4. Bridge old and new – Connect ancient truths to modern life situations so seekers can apply them.
5. Tone & style – Speak with compassion, clarity, patience, depth in knowledge and not just generic, and authority, embodying the voice of a wise spiritual teacher.
6. Mission – Help seekers find guidance, clarity, and transformation through the insights of Shruti and Smriti.
7. crisp – keep it crisp and not very lengthy text for the user to read
`},
                },
            },
        }


	reader := bufio.NewReader(os.Stdin)

	fmt.Println("💬 Gemini Chat (type 'exit' to quit)")
	for {
		// --- User input ---
		fmt.Println()
		fmt.Print("You: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)
		if strings.ToLower(userInput) == "exit" {
			fmt.Println("👋 Goodbye!")
			break
		}

		// Add user message to content
		content = append(content, &genai.Content{
			Role: genai.RoleUser,
			Parts: []*genai.Part{
				{Text: userInput},
			},
		})

		// --- Model response ---
		result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", content, nil)
		if err != nil {
			log.Println("Error:", err)
			continue
		}

		// Print response
		reply := result.Text()

		fmt.Println()
		fmt.Println("AI:", reply)

		// Add AI message to content so context is preserved
		content = append(content, &genai.Content{
			Role: genai.RoleModel,
			Parts: []*genai.Part{
				{Text: reply},
			},
		})
	}
}

