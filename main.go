package main

import (
	"log"
	"os"

	"github.com/dhruv304c2/ai-guru-be.git/service"
	"github.com/dhruv304c2/ai-guru-be.git/test"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Please provide an argument. Use --CLI for the chat CLI or --Service for the HTTP service.")
		return
	}

	switch os.Args[1] {
	case "--Service":
		if err := service.Start(":8080"); err != nil {
			log.Fatalf("HTTP service encountered an error: %v", err)
		}
	case "--CLI":
		test.RunChatCLI()
	default:
		log.Printf("Unknown argument %q. Use --CLI for the chat CLI or --Service for the HTTP service.\n", os.Args[1])
	}
}
