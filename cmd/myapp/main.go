package main

import (
	"context"
	"goproduct/internal/agent"
	"goproduct/internal/chat"
	"goproduct/internal/common"
	"goproduct/internal/entity"
	"goproduct/internal/knowledge"
	"goproduct/internal/llm"
	"goproduct/internal/logging"
	"goproduct/internal/messaging"
	"goproduct/internal/tracing"
	"io"
	"os"
	"time"
)

// RunCLIChatApp runs the CLI chat app with the given input/output streams.
func RunCLIChatApp(in io.Reader, out io.Writer) error {
	ctx := context.Background()

	logger := logging.File("./data/app.log", true)
	defer logger.Close()
	logging.Init(logger)

	// Determine if we're in test mode by checking if input/output are not the standard streams
	isTestMode := in != os.Stdin || out != os.Stdout

	var enhancedTracer *tracing.EnhancedTracer
	var err error

	if isTestMode {
		// Use in-memory tracer for tests to avoid file dependency
		enhancedTracer = tracing.NewMemoryTracer()
	} else {
		// Create directories if they don't exist
		os.MkdirAll("./data", 0755)

		// Use file tracer for normal operation
		enhancedTracer, err = tracing.CreateFileTracer(
			"./data/trace.log",
			5*time.Second,
			4096,
		)
		if err != nil {
			return err
		}
	}
	defer enhancedTracer.Close()

	logging.Get().Info("Application started")
	enhancedTracer.Info("Application started")

	messageBus := messaging.NewMemoryMessageBus()
	enhancedTracer.Info("Message bus created")

	runtime, err := common.NewRuntimeContext(common.RuntimeOptions{
		MessageBus: messageBus,
	})
	if err != nil {
		return err
	}
	enhancedTracer.Info("Runtime context created")

	// Check environment variables for LLM type
	var languageModel llm.LanguageModel
	llmType := os.Getenv("LLM_TYPE")

	switch llmType {
	case "echo":
		// Create an Echo LLM
		echoConfig := &llm.EchoConfig{}
		languageModel, err = llm.NewLLM(ctx, echoConfig)
		if err != nil {
			return err
		}
		enhancedTracer.Info("EchoLLM created with delay from env LLM_DELAY")

	case "exception":
		// Create an Exception LLM
		exceptionConfig := &llm.ExceptionConfig{}
		languageModel, err = llm.NewLLM(ctx, exceptionConfig)
		if err != nil {
			return err
		}
		enhancedTracer.Info("ExceptionLLM created with delay from env LLM_DELAY")

	default:
		// Default to LM Studio LLM
		languageModel, err = llm.NewLMStudioLLM("http://localhost:1234/v1",
			llm.WithLMStudioModel("gemma-3-4b-it"),
			llm.WithLMStudioTemperature(0.7),
			llm.WithLMStudioMaxTokens(4096),
			llm.WithLMStudioTimeout(60),
			llm.WithLMStudioTopP(0.9),
			llm.WithLMStudioPresencePenalty(0.0),
			llm.WithLMStudioFrequencyPenalty(0.0),
		)
		if err != nil {
			return err
		}
		enhancedTracer.Info("LMStudio LLM created")
	}
	enhancedTracer.Info("LLM created")

	store, err := knowledge.NewFileStore("./data/memories.json")
	if err != nil {
		return err
	}
	err = store.Open()
	if err != nil {
		return err
	}
	defer store.Close()

	runtime.SetMemory(store)
	enhancedTracer.Info("Memory store created and added to runtime context")

	store.AddRecord(knowledge.Entry{
		ID:          "1",
		Category:    knowledge.CategoryFact,
		ContentType: knowledge.ContentTypeText,
		Content:     []byte("This is a fact."),
		Importance:  knowledge.ImportanceHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(30 * time.Minute),
		SourceID:    "1",
		SourceType:  "chat",
		OwnerID:     "1",
		OwnerType:   "agent",
		SubjectIDs:  []string{"1"},
		SubjectType: "agent",
		Tags:        []string{"fact"},
		References:  []knowledge.Reference{},
		Metadata:    map[string]string{"source": "chat"},
	})
	err = store.Flush()
	if err != nil {
		return err
	}

	persona := agent.Persona{
		Name: "Andy",
		Role: "Assistant",
		Type: "Text",
		LanguageModels: agent.LanguageModels{
			Default: languageModel,
		},
		SystemPrompt: `
You are an AI Product Owner for a software company that creates websites, HTTP REST services, Android apps, iOS apps, Windows apps, and macOS apps. The CEO is your primary human stakeholder.

# Responsibilities
- Gather and clarify requirements.
- Create and maintain product roadmaps, plans, and specifications.
- Coordinate across departments to ensure alignment.
- Prioritize the product backlog for maximum value.
- Provide strategic, outcome-focused guidance.

# Communication & Tone
- Greet casually (e.g., “Hey, what’s up?”).
- Keep replies short, warm, and to the point—like a helpful teammate.
- Avoid formal or overly detailed language.
- Never use vulgar or insulting language.
- Stay friendly, polite, and adaptive to feedback.

# Scope & Limitations
- Focus strictly on product development topics.
- Politely decline requests unrelated to product development (e.g., weather updates, math solutions, personal opinions unrelated to the product).
- When greeted informally (e.g., “Hello” or “Hey”), respond in a brief, friendly way. If the user asks about or references product matters, respond with strategic, product-focused guidance.
`,
	}

	agentInstance := agent.NewAgent(persona)
	enhancedTracer.Info("Agent created")

	productAgent := entity.NewProductAgentEntity(agentInstance, messageBus)
	enhancedTracer.Info("Product agent entity created: %s (%s)", productAgent.Name(), productAgent.ID())

	humanaEntity := entity.NewCliHumanEntity("User", messageBus)
	enhancedTracer.Info("Human entity created: %s (%s)", humanaEntity.Name(), humanaEntity.ID())

	err = productAgent.Start(ctx)
	if err != nil {
		enhancedTracer.Error("Failed to start product agent: %v", err)
		return err
	}
	enhancedTracer.Info("Product agent started")

	chatInterface := chat.NewEnhancedChat(
		humanaEntity,
		productAgent,
		messageBus,
		enhancedTracer,
	)

	// Set test mode in the chat interface
	chatInterface.IsTestMode = isTestMode
	enhancedTracer.Info("Enhanced chat interface created (isTestMode=%v)", isTestMode)
	enhancedTracer.Info("Enhanced chat interface created")

	// Start the chat interface with custom IO if supported
	if ci, ok := interface{}(chatInterface).(interface {
		StartWithIO(io.Reader, io.Writer) error
	}); ok {
		return ci.StartWithIO(in, out)
	}
	chatInterface.Start()
	return nil
}

func main() {
	_ = RunCLIChatApp(os.Stdin, os.Stdout)
}
