package messaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessageBusEdgeCases tests various edge cases and error conditions
func TestMessageBusEdgeCases(t *testing.T) {
	// Test message handler errors
	t.Run("Message handler errors", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Create an entity with a handler that returns an error
		errorMessage := "intentional error from handler"

		received := make(chan error, 1)
		bus.Subscribe("errorEntity", func(msg Message) error {
			err := errors.New(errorMessage)
			received <- err
			return err
		})

		// Send a message to the entity
		msg := NewTextMessage("sender", []string{"errorEntity"}, "Test message")
		err := bus.Publish(msg)
		assert.NoError(t, err, "Publishing should succeed even if handler will return error")

		// Verify the handler error was received
		select {
		case err := <-received:
			assert.Equal(t, errorMessage, err.Error(), "Should get the expected error from handler")
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for message handler to execute")
		}
	})

	// Test message handler panic recovery
	t.Run("Message handler panic", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Create an entity with a handler that panics
		bus.Subscribe("panicEntity", func(msg Message) error {
			// This will panic but should be caught by our recovery mechanism
			panic("intentional panic in handler")
		})

		// Send a message to the entity - this should not crash the test
		msg := NewTextMessage("sender", []string{"panicEntity"}, "Test message")
		err := bus.Publish(msg)
		assert.NoError(t, err, "Publishing should succeed even if handler will panic")

		// Give the goroutine time to execute
		time.Sleep(100 * time.Millisecond)

		// The test passing means that the panic didn't crash the entire program
		// which is what we want to verify here

		// Note: In a real implementation, we'd want to add panic recovery to the handlers
		// For now we're just verifying the current behavior
	})

	// Test non-existent recipients
	t.Run("Non-existent recipients", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Send a message to a non-existent entity
		msg := NewTextMessage("sender", []string{"nonExistentEntity"}, "Test message")
		err := bus.Publish(msg)
		assert.NoError(t, err, "Publishing to a non-existent entity should not error")

		// Send a message to a non-existent group
		msg = NewTextMessage("sender", []string{"nonExistentGroup"}, "Test message")
		err = bus.Publish(msg)
		assert.NoError(t, err, "Publishing to a non-existent group should not error")
	})

	// Test nil or empty values
	t.Run("Nil and empty values", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Test with nil recipients
		msg := Message{
			ID:          "test",
			SenderID:    "sender",
			Recipients:  nil,
			ContentType: ContentTypeText,
			Content:     []byte("test"),
		}
		err := bus.Publish(msg)
		assert.NoError(t, err, "Publishing with nil recipients should not error")

		// Test with empty sender
		msg = NewTextMessage("", []string{"recipient"}, "Test message")
		err = bus.Publish(msg)
		assert.NoError(t, err, "Publishing with empty sender should not error")

		// Test with nil content
		msg = Message{
			ID:          "test",
			SenderID:    "sender",
			Recipients:  []string{"recipient"},
			ContentType: ContentTypeText,
			Content:     nil,
		}
		err = bus.Publish(msg)
		assert.NoError(t, err, "Publishing with nil content should not error")

		// Test empty group creation
		err = bus.CreateGroup("emptyGroup", "Empty Group", nil)
		assert.NoError(t, err, "Creating a group with nil members should not error")

		members, err := bus.GetGroupMembers("emptyGroup")
		assert.NoError(t, err, "Getting members of an empty group should not error")
		assert.Empty(t, members, "Empty group should have no members")

		// Test with nil handler (this should error)
		err = bus.Subscribe("entity", nil)
		assert.Error(t, err, "Subscribing with nil handler should error")
	})

	// Test group edge cases
	t.Run("Group edge cases", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Create a group
		err := bus.CreateGroup("group1", "Test Group", []string{"member1"})
		assert.NoError(t, err, "Creating a group should succeed")

		// Try creating a duplicate group
		err = bus.CreateGroup("group1", "Duplicate Group", []string{"member2"})
		assert.Error(t, err, "Creating a duplicate group should error")

		// Remove a member that doesn't exist
		err = bus.RemoveFromGroup("group1", "nonExistentMember")
		assert.NoError(t, err, "Removing a non-existent member should not error")

		// Add to a group that doesn't exist
		err = bus.AddToGroup("nonExistentGroup", "member1")
		assert.Error(t, err, "Adding to a non-existent group should error")

		// Get members of a non-existent group
		_, err = bus.GetGroupMembers("nonExistentGroup")
		assert.Error(t, err, "Getting members of a non-existent group should error")
	})

	// Test JSON messages
	t.Run("JSON messages", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Create test data
		type TestData struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		testData := TestData{Name: "test", Value: 42}
		jsonData, err := json.Marshal(testData)
		require.NoError(t, err, "JSON marshaling should succeed")

		// Subscribe a recipient that expects JSON
		received := make(chan Message, 1)
		bus.Subscribe("jsonReceiver", func(msg Message) error {
			received <- msg
			return nil
		})

		// Send a JSON message
		msg := NewJSONMessage("sender", []string{"jsonReceiver"}, jsonData)
		err = bus.Publish(msg)
		assert.NoError(t, err, "Publishing JSON message should succeed")

		// Verify the message was received with correct content type and data
		select {
		case receivedMsg := <-received:
			assert.Equal(t, ContentTypeJSON, receivedMsg.ContentType, "Content type should be JSON")

			// Try to unmarshal and verify the data
			var receivedData TestData
			err := json.Unmarshal(receivedMsg.Content, &receivedData)
			assert.NoError(t, err, "Should be able to unmarshal the JSON data")
			assert.Equal(t, testData.Name, receivedData.Name, "JSON data should match original")
			assert.Equal(t, testData.Value, receivedData.Value, "JSON data should match original")
		case <-time.After(time.Second):
			t.Fatal("Timed out waiting for JSON message")
		}
	})

	// Test concurrent operations
	t.Run("Concurrent operations", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Set up some entities
		entityCount := 10
		messageCount := 20

		var wg sync.WaitGroup

		// Track received messages per entity
		receivedCounts := make([]int, entityCount)
		countLock := sync.Mutex{}

		// Subscribe multiple entities
		for i := 0; i < entityCount; i++ {
			entityID := fmt.Sprintf("entity%d", i)
			entityIndex := i // Capture loop variable

			// Create handler that tracks received messages
			handler := func(msg Message) error {
				countLock.Lock()
				receivedCounts[entityIndex]++
				countLock.Unlock()
				return nil
			}

			err := bus.Subscribe(entityID, handler)
			assert.NoError(t, err, "Subscribing should succeed")
		}

		// Concurrently publish messages to all entities
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < messageCount; i++ {
				// Create recipients list with all entities
				recipients := make([]string, entityCount)
				for j := 0; j < entityCount; j++ {
					recipients[j] = fmt.Sprintf("entity%d", j)
				}

				msg := NewTextMessage("sender", recipients, fmt.Sprintf("Message %d", i))
				err := bus.Publish(msg)
				assert.NoError(t, err, "Publishing should succeed")
			}
		}()

		// Concurrently unsubscribe and resubscribe some entities
		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 0; i < 5; i++ {
				// Pick an entity to unsubscribe
				entityID := fmt.Sprintf("entity%d", i)

				// Unsubscribe
				err := bus.Unsubscribe(entityID)
				assert.NoError(t, err, "Unsubscribing should succeed")

				// Sleep a bit to allow some messages to be missed
				time.Sleep(10 * time.Millisecond)

				// Resubscribe with the same handler
				entityIndex := i
				handler := func(msg Message) error {
					countLock.Lock()
					receivedCounts[entityIndex]++
					countLock.Unlock()
					return nil
				}

				err = bus.Subscribe(entityID, handler)
				assert.NoError(t, err, "Resubscribing should succeed")
			}
		}()

		// Concurrently create and modify groups
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create a group
			groupID := "testGroup"
			err := bus.CreateGroup(groupID, "Test Group", []string{"entity0", "entity1"})
			assert.NoError(t, err, "Creating group should succeed")

			// Modify group membership
			for i := 0; i < 10; i++ {
				// Add an entity
				entityID := fmt.Sprintf("entity%d", i)
				err := bus.AddToGroup(groupID, entityID)

				// First time for entities 0 and 1 will return "already exists" error
				if i > 1 {
					assert.NoError(t, err, "Adding to group should succeed")
				}

				// Sleep a bit
				time.Sleep(5 * time.Millisecond)

				// Send a message to the group
				msg := NewTextMessage("sender", []string{groupID}, fmt.Sprintf("Group message %d", i))
				err = bus.Publish(msg)
				assert.NoError(t, err, "Publishing to group should succeed")
			}
		}()

		// Wait for all concurrent operations to complete
		wg.Wait()

		// Allow time for all message handlers to complete
		time.Sleep(100 * time.Millisecond)

		// Check that messages were delivered
		countLock.Lock()
		defer countLock.Unlock()

		totalReceived := 0
		for i, count := range receivedCounts {
			t.Logf("Entity %d received %d messages", i, count)
			totalReceived += count
		}

		// We can't know exactly how many messages each entity received due to the
		// concurrent unsubscribe/resubscribe, but we should have a substantial number overall
		assert.Greater(t, totalReceived, 0, "Some messages should have been received")
	})

	// Test tracer integration
	t.Run("Tracer operation", func(t *testing.T) {
		bus := NewMemoryMessageBus()

		// Check default tracer
		tracer := bus.GetTracer()
		assert.NotNil(t, tracer, "Default tracer should not be nil")

		// Set a nil tracer (should be handled gracefully)
		bus.SetTracer(nil)

		// Sending a message with nil tracer should not panic
		msg := NewTextMessage("sender", []string{"recipient"}, "Test message")
		err := bus.Publish(msg)
		assert.NoError(t, err, "Publishing with nil tracer should not error")

		// Restore a valid tracer
		bus.SetTracer(tracer)
	})
}

// TestRecursiveGroups tests nested group behavior
func TestRecursiveGroups(t *testing.T) {
	bus := NewMemoryMessageBus()

	// Create entities
	entityReceived := make(map[string]chan Message)
	for i := 1; i <= 5; i++ {
		entityID := fmt.Sprintf("entity%d", i)
		entityReceived[entityID] = make(chan Message, 10)

		// Create closure to capture correct entityID for the handler
		func(id string) {
			bus.Subscribe(id, func(msg Message) error {
				entityReceived[id] <- msg
				return nil
			})
		}(entityID)
	}

	// Create a hierarchical group structure
	// Group1 contains entity1, entity2
	// Group2 contains entity3, entity4
	// Group3 contains entity5, Group1, Group2
	bus.CreateGroup("group1", "Group 1", []string{"entity1", "entity2"})
	bus.CreateGroup("group2", "Group 2", []string{"entity3", "entity4"})
	bus.CreateGroup("group3", "Group 3", []string{"entity5", "group1", "group2"})

	// Send a message to the top-level group
	msg := NewTextMessage("sender", []string{"group3"}, "Hierarchical message")
	err := bus.Publish(msg)
	assert.NoError(t, err, "Publishing to group should succeed")

	// Currently our implementation doesn't support recursive group resolution
	// This test demonstrates the current behavior

	// Only entity5 should receive the message directly
	// Group1 and Group2 will be treated as entities, not as groups

	// Check entity5 received the message
	select {
	case receivedMsg := <-entityReceived["entity5"]:
		text, err := receivedMsg.TextContent()
		assert.NoError(t, err, "Getting text content should succeed")
		assert.Equal(t, "Hierarchical message", text, "Entity5 should receive the message")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Entity5 should have received the message")
	}

	// The entities in the subgroups should not receive the message with the current implementation
	for i := 1; i <= 4; i++ {
		entityID := fmt.Sprintf("entity%d", i)
		select {
		case <-entityReceived[entityID]:
			t.Fatalf("Entity %s should not have received the message with current implementation", entityID)
		case <-time.After(100 * time.Millisecond):
			// This is expected with current implementation
		}
	}

	// This test shows a limitation in our current implementation
	// A real improvement would be to implement recursive group resolution
}

// TestMessageValidation tests validation of message fields
func TestMessageValidation(t *testing.T) {
	// Test TextContent with wrong content type
	t.Run("Wrong content type", func(t *testing.T) {
		msg := Message{
			ID:          "test",
			SenderID:    "sender",
			Recipients:  []string{"recipient"},
			ContentType: ContentTypeJSON,
			Content:     []byte("This is not JSON"),
		}

		_, err := msg.TextContent()
		assert.Error(t, err, "Getting text content with wrong content type should error")
	})

	// Test message creation with empty fields
	t.Run("Empty fields", func(t *testing.T) {
		// Empty sender
		msg := NewTextMessage("", []string{"recipient"}, "Test message")
		assert.Empty(t, msg.SenderID, "Message should have empty sender")
		assert.NotEmpty(t, msg.ID, "Message should still have an ID")

		// Empty recipients
		msg = NewTextMessage("sender", []string{}, "Test message")
		assert.Empty(t, msg.Recipients, "Message should have empty recipients")

		// Empty content
		msg = NewTextMessage("sender", []string{"recipient"}, "")
		assert.Empty(t, string(msg.Content), "Message should have empty content")
	})
}

// TestMessageBusThreadSafety verifies the message bus is thread-safe
func TestMessageBusThreadSafety(t *testing.T) {
	bus := NewMemoryMessageBus()

	// Number of concurrent operations
	goroutines := 100
	operations := 100

	// Run various operations concurrently
	var wg sync.WaitGroup
	errs := make(chan error, goroutines*operations)

	// Helper to report errors
	reportErr := func(err error) {
		if err != nil {
			errs <- err
		}
	}

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operations; j++ {
				// Mix different operations
				switch j % 5 {
				case 0: // Subscribe
					entityID := fmt.Sprintf("entity-%d-%d", id, j)
					err := bus.Subscribe(entityID, func(msg Message) error {
						return nil
					})
					reportErr(err)

				case 1: // Unsubscribe
					entityID := fmt.Sprintf("entity-%d-%d", id, j-1)
					err := bus.Unsubscribe(entityID)
					reportErr(err)

				case 2: // Create group
					groupID := fmt.Sprintf("group-%d-%d", id, j)
					err := bus.CreateGroup(groupID, "Test Group", nil)
					reportErr(err)

				case 3: // Add to group
					groupID := fmt.Sprintf("group-%d-%d", id, j-1)
					entityID := fmt.Sprintf("entity-%d-%d", id, j)
					err := bus.AddToGroup(groupID, entityID)
					// Ignore errors from non-existent groups
					if err != nil && err.Error() != fmt.Sprintf("group with ID %s does not exist", groupID) {
						reportErr(err)
					}

				case 4: // Publish
					recipients := []string{fmt.Sprintf("entity-%d-%d", id, j)}
					msg := NewTextMessage(fmt.Sprintf("sender-%d", id), recipients, "Test message")
					err := bus.Publish(msg)
					reportErr(err)
				}
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(errs)

	// Check for any errors
	for err := range errs {
		t.Errorf("Concurrent operation error: %v", err)
	}
}
