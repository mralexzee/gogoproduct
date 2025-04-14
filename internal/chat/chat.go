package chat

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"goproduct/internal/agent"

	"github.com/manifoldco/promptui"
)

// Command represents a special chat command
type Command struct {
	Name        string
	Description string
	Handler     func() string
}

// Chat represents the chat interface
type Chat struct {
	commands    map[string]Command
	prompt      *promptui.Prompt
	agent       *agent.Agent
	ctx         context.Context
	cancel      context.CancelFunc
	pendingMsgs map[string]bool // Track messages awaiting response by ID
	responses   chan struct{}   // Just a signal channel for new responses
}

// NewChat creates a new chat interface with an agent
func NewChat(agent *agent.Agent) *Chat {
	ctx, cancel := context.WithCancel(context.Background())
	chat := &Chat{
		commands:    make(map[string]Command),
		agent:       agent,
		ctx:         ctx,
		cancel:      cancel,
		pendingMsgs: make(map[string]bool),
		responses:   make(chan struct{}, 10), // Signal channel
	}

	// Start the agent
	agent.Start(ctx)

	// Configure the prompt with multiline support
	prompt := promptui.Prompt{
		Label:       "user",
		AllowEdit:   true,
		HideEntered: false,
	}

	chat.prompt = &prompt

	// Register default commands
	chat.registerCommands()

	return chat
}

// registerCommands registers the default commands
func (c *Chat) registerCommands() {
	c.commands["help()"] = Command{
		Name:        "help()",
		Description: "Show available commands",
		Handler: func() string {
			var sb strings.Builder
			sb.WriteString("Available commands:\n")
			for _, cmd := range c.commands {
				sb.WriteString(fmt.Sprintf("  %s - %s\n", cmd.Name, cmd.Description))
			}
			return sb.String()
		},
	}

	c.commands["exit()"] = Command{
		Name:        "exit()",
		Description: "Exit the application",
		Handler: func() string {
			// Clean up resources before exit
			c.cancel()
			c.agent.Stop()
			fmt.Println("Goodbye!")
			os.Exit(0)
			return ""
		},
	}

	c.commands["quit()"] = Command{
		Name:        "quit()",
		Description: "Exit the application",
		Handler: func() string {
			// Clean up resources before exit
			c.cancel()
			c.agent.Stop()
			fmt.Println("Goodbye!")
			os.Exit(0)
			return ""
		},
	}

	c.commands["now()"] = Command{
		Name:        "now()",
		Description: "Show current date and time",
		Handler: func() string {
			return fmt.Sprintf("Current time: %s", time.Now().Format("Monday, January 2, 2006 at 3:04:05 PM MST"))
		},
	}
}

// displayPendingMessages shows status of pending messages
func (c *Chat) displayPendingMessages() {
	if len(c.pendingMsgs) > 0 {
		fmt.Println("Pending messages:", len(c.pendingMsgs))
	}
}

// processResponse handles an incoming message
func (c *Chat) processResponse(msg agent.Message) {
	go func() {
		// Get a random ID for reference
		msgId := msg.Id

		// Wait for response with timeout
		select {
		case <-c.ctx.Done():
			return
		case response := <-msg.ResponseReady:
			// Print the response immediately
			fmt.Printf("Agent [%s]: %s\n\n", msgId[:8], response.Content)

			// Mark message as done
			delete(c.pendingMsgs, msgId)

			// Signal that we got a response
			select {
			case c.responses <- struct{}{}:
			default: // Don't block if channel full
			}

		case <-time.After(30 * time.Second): // Longer timeout
			fmt.Printf("Response timed out for message: %s\n", msgId[:8])
			delete(c.pendingMsgs, msgId)
		}
	}()
}

// Start begins the chat interface
func (c *Chat) Start() {
	// Setup signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nGoodbye!")
		// Clean up resources before exit
		c.cancel()
		c.agent.Stop()
		os.Exit(0)
	}()

	fmt.Println("Welcome to the Chat Interface!")
	fmt.Println("Type help() for available commands")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	username := "User" // Default username

	// Display pending messages status periodically
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.displayPendingMessages()
			case <-c.responses:
				// Just drain the channel
			}
		}
	}()

	// Main input loop
	for {
		// Get input using promptui
		result, err := c.prompt.Run()

		if err != nil {
			fmt.Printf("Prompt failed: %v\n", err)
			return
		}

		// Check if input is a command
		trimmedInput := strings.TrimSpace(result)
		if command, exists := c.commands[trimmedInput]; exists {
			response := command.Handler()
			if response != "" {
				fmt.Println(response)
			}
		} else if trimmedInput != "" {
			// Send message to agent without waiting for response
			msg := c.agent.Chat(username, trimmedInput)

			// Add to pending messages
			c.pendingMsgs[msg.Id] = true

			// Show the message ID so user can track it
			fmt.Printf("Message sent [%s]\n", msg.Id[:8])

			// Process the response asynchronously
			c.processResponse(msg)
		}
	}
}
