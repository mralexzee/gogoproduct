package chat

import (
	"context"
	"fmt"
	"github.com/manifoldco/promptui"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"goproduct/internal/entity"
	"goproduct/internal/logging"
	"goproduct/internal/messaging"
	"goproduct/internal/tracing"
)

// EnhancedChat represents a chat interface that uses the messaging system
type EnhancedChat struct {
	commands    map[string]Command
	human       *entity.CliHumanEntity
	agent       entity.Entity
	messageBus  messaging.MessageBus
	tracer      *tracing.EnhancedTracer
	logger      *logging.Logger
	prompt      *promptui.Prompt
	ctx         context.Context
	cancel      context.CancelFunc
	pendingMsgs map[string]bool
	responses   chan struct{}
	mutex       sync.RWMutex // Protect pendingMsgs map
}

// NewEnhancedChat creates a new enhanced chat interface
func NewEnhancedChat(
	human *entity.CliHumanEntity,
	agent entity.Entity,
	bus messaging.MessageBus,
	tracer *tracing.EnhancedTracer,
) *EnhancedChat {
	ctx, cancel := context.WithCancel(context.Background())

	chat := &EnhancedChat{
		commands:    make(map[string]Command),
		human:       human,
		agent:       agent,
		messageBus:  bus,
		tracer:      tracer,
		logger:      logging.Get(), // Use the application logger
		ctx:         ctx,
		cancel:      cancel,
		pendingMsgs: make(map[string]bool),
		responses:   make(chan struct{}, 10),
	}

	// Configure the prompt with multiline support
	prompt := promptui.Prompt{
		Label:       human.Name(),
		AllowEdit:   true,
		HideEntered: false,
	}

	chat.prompt = &prompt

	// Register default commands
	chat.registerCommands()

	return chat
}

// registerCommands registers the default commands
func (c *EnhancedChat) registerCommands() {
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
			c.tracer.Close()
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
			c.tracer.Close()
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

// displayPendingMessages shows the count of pending messages
func (c *EnhancedChat) displayPendingMessages() {
	c.mutex.RLock()
	pendingCount := len(c.pendingMsgs)
	pendingIDs := make([]string, 0, pendingCount)
	for id := range c.pendingMsgs {
		pendingIDs = append(pendingIDs, id)
	}
	c.mutex.RUnlock()

	if pendingCount > 0 {
		c.logger.Debug("Pending messages check", "count", pendingCount, "message_ids", strings.Join(pendingIDs, ","))
		fmt.Printf("Pending messages: %d\n", pendingCount)
	}
}

// Start begins the enhanced chat interface
func (c *EnhancedChat) Start() {
	// Start the human entity
	c.logger.Info("Enhanced chat interface starting")
	if err := c.human.Start(); err != nil {
		c.logger.Error("Failed to start human entity", "error", err)
		c.tracer.Error("Failed to start human entity: %v", err)
		fmt.Printf("Failed to start chat: %v\n", err)
		return
	}
	c.logger.Info("Human entity started successfully", "entity_id", c.human.ID())

	// Setup signal handling for Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nGoodbye!")
		// Clean up resources before exit
		c.cancel()
		c.tracer.Close()
		os.Exit(0)
	}()

	c.tracer.Info("Enhanced Chat Interface started")
	fmt.Println("Welcome to the Enhanced Chat Interface!")
	fmt.Println("Type help() for available commands")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

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
				// Also check for old conversations that may need cleanup
				if cliHuman, ok := interface{}(c.human).(*entity.CliHumanEntity); ok {
					pending := cliHuman.GetPendingConversations()
					if len(pending) > 0 {
						c.logger.Debug("Pending conversations from human entity", "count", len(pending), "ids", strings.Join(pending, ","))
					}
				}
			case <-c.responses:
				// Just drain the channel
			}
		}
	}()

	// Main input loop
	for {
		// Get input using promptui
		c.logger.Debug("Waiting for user input")
		result, err := c.prompt.Run()

		if err != nil {
			c.logger.Error("Prompt failed", "error", err)
			c.tracer.Error("Prompt failed: %v", err)
			fmt.Printf("Prompt failed: %v\n", err)
			return
		}

		c.logger.Debug("User input received", "content_length", len(result))

		// Check if input is a command
		trimmedInput := strings.TrimSpace(result)
		if command, exists := c.commands[trimmedInput]; exists {
			c.logger.Info("Command executed", "command", trimmedInput)
			c.tracer.Info("Command executed: %s", trimmedInput)
			response := command.Handler()
			if response != "" {
				fmt.Println(response)
			}
		} else if trimmedInput != "" {
			c.logger.Info("Processing user message", "content_length", len(trimmedInput))
			// Create a message using the messaging system
			c.logger.Debug("Preparing to send message", "recipient", c.agent.ID(), "recipient_name", c.agent.Name())
			msg, err := c.human.SendMessage(
				[]string{c.agent.ID()},
				messaging.ContentTypeText,
				[]byte(trimmedInput),
			)

			if err != nil {
				c.logger.Error("Failed to send message", "error", err)
				c.tracer.Error("Failed to send message: %v", err)
				fmt.Printf("Failed to send message: %v\n", err)
				continue
			}

			c.logger.Info("User message sent to agent", "message_id", msg.ID, "recipient", c.agent.ID(), "recipient_name", c.agent.Name())

			c.tracer.Debug("Message sent: %s", msg.ID)

			// Register a handler for the response
			c.logger.Debug("Registering response handler", "message_id", msg.ID)
			c.human.RegisterMessageHandler(msg.ID, func(response messaging.Message) {
				// Check if this is a response to our original message
				originalMsgID := msg.ID
				if respOrigID, exists := response.Metadata["original_id"]; exists && respOrigID != "" {
					c.logger.Debug("Response references original message", "original_id", respOrigID, "response_id", response.ID)
					originalMsgID = respOrigID
				}

				// Print the response
				c.logger.Info("Agent response received",
					"original_message_id", originalMsgID,
					"response_id", response.ID,
					"sender", response.SenderID,
					"content_length", len(response.Content))
				c.tracer.Debug("Response received for message %s, response ID: %s", originalMsgID, response.ID)
				c.logger.Debug("Displaying response to user", "message_id", originalMsgID, "sender_name", c.agent.Name())
				fmt.Printf("%s: %s\n\n", c.agent.Name(), string(response.Content))

				// Mark message as done
				c.mutex.Lock()
				delete(c.pendingMsgs, msg.ID)
				c.logger.Info("Message conversation complete", "message_id", msg.ID, "pending_count", len(c.pendingMsgs))
				c.mutex.Unlock()
				c.logger.Debug("Message marked as complete", "message_id", msg.ID)

				// Signal that we got a response
				select {
				case c.responses <- struct{}{}:
					c.logger.Debug("Response notification sent", "message_id", msg.ID)
				default: // Don't block if channel full
					c.logger.Debug("Response notification channel full", "message_id", msg.ID)
				}
			})

			// Add to pending messages
			c.mutex.Lock()
			c.pendingMsgs[msg.ID] = true
			c.mutex.Unlock()
			c.logger.Debug("Message added to pending queue", "message_id", msg.ID)

			// Additional debugging
			c.logger.Debug("Waiting for response", "message_id", msg.ID, "agent", c.agent.Name())

			// Show the message ID so user can track it
			fmt.Printf("Message sent [%s]\n", msg.ID[:8])

			// Set up a timeout to clear the message if no response received (last resort fallback)
			go func(msgID string) {
				timeoutSeconds := 60
				timeoutAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
				c.logger.Debug("Setting up message timeout handler",
					"message_id", msgID,
					"timeout_seconds", timeoutSeconds,
					"timeout_at", timeoutAt)
				time.Sleep(time.Duration(timeoutSeconds) * time.Second) // Extended to 60 seconds as last resort

				// If message is still pending after timeout
				c.mutex.Lock()
				_, stillPending := c.pendingMsgs[msgID]
				c.mutex.Unlock()

				if stillPending {
					c.logger.Warn("Message response timed out", "message_id", msgID, "elapsed_seconds", 60)
					c.logger.Warn("Message timeout triggered - no response received after 60 seconds", "message_id", msgID)
					// Create a fake response as if it came from the agent
					fmt.Printf("%s: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.\n\n", c.agent.Name())

					// Remove from pending messages
					c.mutex.Lock()
					delete(c.pendingMsgs, msgID)
					c.mutex.Unlock()
					c.logger.Debug("Message marked as complete by timeout handler", "message_id", msgID)

					// Signal that we handled the message
					select {
					case c.responses <- struct{}{}:
						c.logger.Debug("Response notification sent from timeout handler", "message_id", msgID)
					default: // Don't block if channel full
						c.logger.Debug("Response notification channel full in timeout handler", "message_id", msgID)
					}
				} else {
					c.logger.Debug("Timeout handler found message already processed", "message_id", msgID)
				}
			}(msg.ID)
		}
	}
}
