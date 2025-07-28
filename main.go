package main

import (
	"fmt"
	"log"
	"os"

	"github.com/andrinoff/email-cli/config"
	"github.com/andrinoff/email-cli/tui"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		// If config file does not exist, run the login UI
		if os.IsNotExist(err) {
			fmt.Println("No configuration found. Starting setup...")
			newCfg, err := tui.RunLogin()
			if err != nil {
				log.Fatalf("Could not run login UI: %v", err)
			}
			// If user quit the login screen without saving, exit
			if newCfg == nil {
				fmt.Println("\nSetup cancelled. Exiting.")
				os.Exit(0)
			}
			cfg = newCfg
		} else {
			// For any other error loading the config, exit
			log.Fatalf("Could not load config: %v", err)
		}
	}

	// Now that we have a config, let the user choose what to do
	choice, err := tui.RunChoice()
	if err != nil {
		log.Fatalf("Could not run choice UI: %v", err)
	}

	switch choice {
	case "send":
		fmt.Printf("Logged in as %s. Starting email composer...\n", cfg.Email)
		if err := tui.RunComposer(cfg); err != nil {
			log.Fatalf("Application error: %v", err)
		}
	case "inbox":
		fmt.Printf("Logged in as %s. Opening inbox...\n", cfg.Email)
		if err := tui.RunInbox(cfg); err != nil {
			log.Fatalf("Application error: %v", err)
		}
	}
}