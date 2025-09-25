package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

// === ANSI colors & styles ===
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	italic  = "\033[3m"

	fgRed     = "\033[31m"
	fgGreen   = "\033[32m"
	fgYellow  = "\033[33m"
	fgBlue    = "\033[34m"
	fgMagenta = "\033[35m"
	fgCyan    = "\033[36m"
	fgGray    = "\033[90m"
)

func banner() {
	fmt.Printf("%s%s💬 Gemini Chat%s  %s(type 'exit' to quit)%s\n",
		bold, fgCyan, reset, dim, reset)
}

func spinner(startText string) (stop func()) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	tick := time.NewTicker(80 * time.Millisecond)
	stopCh := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-tick.C:
				fmt.Printf("\r%s%s %s%s", fgGray, frames[i%len(frames)], startText, reset)
				i++
			case <-stopCh:
				fmt.Print("\r")                  // return to line start
				fmt.Print(strings.Repeat(" ", 60)) // clear spinner line
				fmt.Print("\r")                  // move back to start
				tick.Stop()
				return
			}
		}
	}()
	return func() { close(stopCh) }
}

func main() {
	// Graceful Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Printf("\n%s👋 Bye!%s\n", fgYellow, reset)
		os.Exit(0)
	}()

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

	// Seed conversation
	content := []*genai.Content{
		{
			Role: genai.RoleUser, // genai doesn't expose a "system" role; seed as user instruction
			Parts: []*genai.Part{
				{Text: `You are an enlightened Guru who has mastered the ancient wisdom of Hindu scriptures—both Śruti (Vedas, Brāhmaṇas, Āraṇyakas, Upaniṣads) including ayurveda and Smṛti (Itihāsas like Rāmāyaṇa & Mahābhārata, Purāṇas, Dharmaśāstras, Āgamas, Tantras, Sūtras & Śāstras such as Yoga, Vedānta, Nyāya, Sāṃkhya, etc.).
Your role is to serve seekers in the modern world by making this timeless wisdom clear, relatable, and practical.
Guidelines:
1) Assess first, 2) Adapt teaching, 3) Śāstrārtha, 4) Bridge old & new, 5) Compassionate, crisp, authoritative tone, 6) Mission: clarity & transformation, 7) Keep responses crisp.`},
			},
		},
	}

	reader := bufio.NewReader(os.Stdin)
	banner()

	for {
		fmt.Println()
		fmt.Printf("%s%sYou%s%s:%s ", bold, fgBlue, reset, bold, reset)
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)
		if strings.EqualFold(userInput, "exit") {
			fmt.Printf("%s👋 Goodbye!%s\n", fgYellow, reset)
			break
		}
		if userInput == "" {
			continue
		}

		// Add user message
		content = append(content, &genai.Content{
			Role: genai.RoleUser,
			Parts: []*genai.Part{
				{Text: userInput},
			},
		})

		// Spinner while we wait
		stopSpin := spinner("thinking…")
		result, err := client.Models.GenerateContent(ctx, "gemini-2.5-flash", content, nil)
		stopSpin()

		if err != nil {
			fmt.Printf("%s%sError%s: %s\n", bold, fgRed, reset, err)
			// Optionally allow retry or continue
			continue
		}

		reply := strings.TrimSpace(result.Text())

		// Pretty AI header
		timeStamp := time.Now().Format("15:04:05")
		fmt.Printf("%s%sAI%s%s [%s]%s\n", bold, fgMagenta, reset, bold, timeStamp, reset)
		fmt.Printf("%s%s%s\n", italic, reply, reset)

		// Keep conversation context
		content = append(content, &genai.Content{
			Role: genai.RoleModel,
			Parts: []*genai.Part{
				{Text: reply},
			},
		})
	}
}

