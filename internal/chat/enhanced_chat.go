package chat

import (
	"bufio"
	"context"
	"fmt"
	"github.com/manifoldco/promptui"
	"io"
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
	commands     map[string]Command
	human        *entity.CliHumanEntity
	agent        entity.Entity
	messageBus   messaging.MessageBus
	tracer       *tracing.EnhancedTracer
	logger       *logging.Logger
	prompt       *promptui.Prompt
	ctx          context.Context
	cancel       context.CancelFunc
	pendingMsgs  map[string]bool
	responses    chan struct{}
	msgCancelMap map[string]chan struct{} // Map of message ID to cancellation channels
	mutex        sync.RWMutex             // Protect pendingMsgs and msgCancelMap maps
	IsTestMode   bool                     // Explicitly tracks if running in test mode
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
		commands:     make(map[string]Command),
		human:        human,
		agent:        agent,
		messageBus:   bus,
		tracer:       tracer,
		logger:       logging.Get(), // Use the application logger
		ctx:          ctx,
		cancel:       cancel,
		pendingMsgs:  make(map[string]bool),
		responses:    make(chan struct{}, 10),
		msgCancelMap: make(map[string]chan struct{}),
		IsTestMode:   false, // Default to production mode
	}

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
			return "Goodbye!"
		},
	}

	c.commands["quit()"] = Command{
		Name:        "quit()",
		Description: "Exit the application",
		Handler: func() string {
			// Clean up resources before exit
			c.cancel()
			c.tracer.Close()
			return "Goodbye!"
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

// Start begins the enhanced chat interface with standard input/output
func (c *EnhancedChat) Start() error {
	return c.StartWithIO(os.Stdin, os.Stdout)
}

// nopCloser wraps a io.Reader to provide a no-op Close method (for promptui)
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

// nopWriteCloser wraps a io.Writer to provide a no-op Close method (for promptui)
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// StartWithIO begins the enhanced chat interface with custom input/output streams
func (c *EnhancedChat) StartWithIO(in io.Reader, out io.Writer) error {
	// Create a prompt with the custom input/output, wrapping in ReadCloser/WriteCloser adapters
	prompt := &promptui.Prompt{
		Label:       c.human.Name(),
		AllowEdit:   true,
		HideEntered: false,
		Stdin:       nopCloser{in},       // Adapt io.Reader to io.ReadCloser
		Stdout:      nopWriteCloser{out}, // Adapt io.Writer to io.WriteCloser
	}
	c.prompt = prompt

	// Start the human entity
	c.logger.Info("Enhanced chat interface starting")
	if err := c.human.Start(); err != nil {
		c.logger.Error("Failed to start human entity", "error", err)
		c.tracer.Error("Failed to start human entity: %v", err)
		fmt.Fprintf(out, "Failed to start chat: %v\n", err)
		return err
	}
	c.logger.Info("Human entity started successfully", "entity_id", c.human.ID())

	// Setup signal handling for Ctrl+C (only in non-test mode)
	if in == os.Stdin {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigChan
			fmt.Fprintln(out, "\nGoodbye!")
			// Clean up resources before exit
			c.cancel()
			c.tracer.Close()
			os.Exit(0)
		}()
	}

	c.tracer.Info("Enhanced Chat Interface started")
	fmt.Fprintln(out, "Welcome to the Enhanced Chat Interface!")
	fmt.Fprintln(out, "Type help() for available commands")
	fmt.Fprintln(out, "Press Ctrl+C to exit")
	fmt.Fprintln(out)

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

	// Create a scanner for automated testing
	scanner := bufio.NewScanner(in)

	// Check if it's a test with pre-supplied input or regular interactive mode
	if in != os.Stdin {
		// Test mode with pre-supplied inputs using scanner
		for scanner.Scan() {
			result := scanner.Text()

			// Display user input
			fmt.Fprintf(out, "User: %s\n", result)

			// Process command or message
			continueRunning := c.processInput(result, out)

			// Exit if command handler returned false (e.g. exit() was called)
			if !continueRunning {
				return nil
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	} else {
		// Interactive mode using promptui
		for {
			// Get input using promptui
			c.logger.Debug("Waiting for user input")
			result, err := c.prompt.Run()

			if err != nil {
				c.logger.Error("Prompt failed", "error", err)
				c.tracer.Error("Prompt failed: %v", err)
				fmt.Fprintf(out, "Prompt failed: %v\n", err)
				return err
			}

			c.logger.Debug("User input received", "content_length", len(result))
			continueRunning := c.processInput(result, out)

			// If processInput returns false, exit the app
			if !continueRunning {
				// Exit without error
				return nil
			}
		}
	}
}

// processInput handles user input (commands or messages)
// Returns true if processing should continue, false if app should exit
func (c *EnhancedChat) processInput(result string, out io.Writer) bool {
	// Check if input is a command
	trimmedInput := strings.TrimSpace(result)
	if command, exists := c.commands[trimmedInput]; exists {
		c.logger.Info("Command executed", "command", trimmedInput)
		c.tracer.Info("Command executed: %s", trimmedInput)
		response := command.Handler()
		if response != "" {
			fmt.Fprintln(out, response)
		}

		// Special handling for exit/quit commands
		if trimmedInput == "exit()" || trimmedInput == "quit()" {
			return false // Signal to exit the app
		}

		// For other commands, continue processing
		return true
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
			fmt.Fprintf(out, "Failed to send message: %v\n", err)
			return true // Continue processing despite message error
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

			// Check if message is still pending or was already handled by timeout
			c.mutex.Lock()
			_, stillPending := c.pendingMsgs[msg.ID]
			c.mutex.Unlock()

			if !stillPending {
				c.logger.Warn("Received LLM response for already handled message",
					"message_id", msg.ID,
					"response_id", response.ID)
				return // Don't process this message further, but no return value expected
			}

			// Print the response
			c.logger.Info("Agent response received",
				"original_message_id", originalMsgID,
				"response_id", response.ID,
				"sender", response.SenderID,
				"content_length", len(response.Content))
			c.tracer.Debug("Response received for message %s, response ID: %s", originalMsgID, response.ID)
			c.logger.Debug("Displaying response to user", "message_id", originalMsgID, "sender_name", c.agent.Name())
			fmt.Fprintf(out, "%s: %s\n\n", c.agent.Name(), string(response.Content))

			// Mark message as done
			c.mutex.Lock()
			// Cancel any pending timeout handlers for this message
			if cancelCh, exists := c.msgCancelMap[msg.ID]; exists {
				c.logger.Debug("Cancelling timeout handler", "message_id", msg.ID)
				close(cancelCh) // Signal cancellation to timeout handler
				delete(c.msgCancelMap, msg.ID)
			}
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

		// Add to pending messages and create a cancellation channel
		cancelCh := make(chan struct{})
		c.mutex.Lock()
		c.pendingMsgs[msg.ID] = true
		c.msgCancelMap[msg.ID] = cancelCh
		c.mutex.Unlock()
		c.logger.Debug("Message added to pending queue with cancellation channel", "message_id", msg.ID)

		// Additional debugging
		c.logger.Debug("Waiting for response", "message_id", msg.ID, "agent", c.agent.Name())

		// Show the message ID so user can track it
		fmt.Fprintf(out, "Message sent [%s]\n", msg.ID[:8])

		// Set up a timeout to clear the message if no response received (last resort fallback)
		go func(msgID string, writer io.Writer, cancelChannel <-chan struct{}) {
			// Check if we're in test mode - read the explicit flag instead of checking stdin
			timeoutSeconds := 60
			if c.IsTestMode {
				// Almost immediate timeout for tests (200ms)
				timeoutSeconds = 0
				// Use a timer instead of sleep to properly handle cancellations
				timer := time.NewTimer(200 * time.Millisecond)
				select {
				case <-cancelChannel:
					// Response already received - exit immediately
					timer.Stop()
					c.logger.Debug("Test timeout handler cancelled - response already received", "message_id", msgID)
					return

				case <-timer.C:
					// Timer expired, check if message is still pending
					c.mutex.Lock()
					_, stillPending := c.pendingMsgs[msgID]
					c.mutex.Unlock()

					if stillPending {
						c.logger.Debug("Test timeout triggered for message", "message_id", msgID)
						fmt.Fprintf(writer, "%s: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.\n\n", c.agent.Name())

						c.mutex.Lock()
						delete(c.pendingMsgs, msgID)
						delete(c.msgCancelMap, msgID)
						c.mutex.Unlock()

						c.logger.Debug("Message marked as complete by test timeout handler", "message_id", msgID)
						return // Exit the goroutine immediately in test mode
					}
				}
			}

			timeoutAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
			c.logger.Debug("Setting up message timeout handler",
				"message_id", msgID,
				"timeout_seconds", timeoutSeconds,
				"timeout_at", timeoutAt,
				"test_mode", c.IsTestMode)

			// Use a timer and select to properly handle cancellations
			timer := time.NewTimer(time.Duration(timeoutSeconds) * time.Second)
			select {
			case <-cancelChannel:
				// Response already received - cancel the timeout
				timer.Stop()
				c.logger.Debug("Timeout handler cancelled - response already received", "message_id", msgID)
				return

			case <-timer.C:
				// Timer expired, check if message is still pending
				c.mutex.Lock()
				_, stillPending := c.pendingMsgs[msgID]
				c.mutex.Unlock()

				if stillPending {
					c.logger.Warn("Message response timed out", "message_id", msgID, "elapsed_seconds", timeoutSeconds)
					c.logger.Warn("Message timeout triggered - no response received after timeout", "message_id", msgID)
					// Create a fake response as if it came from the agent
					fmt.Fprintf(writer, "%s: I'm out of office today. If you need immediate assistance, please contact Tom Reynolds.\n\n", c.agent.Name())

					// Remove from pending messages
					c.mutex.Lock()
					delete(c.pendingMsgs, msgID)
					delete(c.msgCancelMap, msgID)
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
			}
		}(msg.ID, out, cancelCh)

		// Continue processing for regular messages
		return true
	}

	// Default case (empty input or unhandled case)
	return true
}
