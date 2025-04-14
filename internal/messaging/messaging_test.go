package messaging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryMessageBus(t *testing.T) {
	// Create a new message bus
	bus := NewMemoryMessageBus()

	// Test entity IDs
	entity1ID := "entity1"
	entity2ID := "entity2"
	entity3ID := "entity3"

	// Message tracking
	entity1Received := make(chan Message, 10)
	entity2Received := make(chan Message, 10)
	entity3Received := make(chan Message, 10)

	// Subscribe entities to the bus
	err := bus.Subscribe(entity1ID, func(msg Message) error {
		entity1Received <- msg
		return nil
	})
	assert.NoError(t, err)

	err = bus.Subscribe(entity2ID, func(msg Message) error {
		entity2Received <- msg
		return nil
	})
	assert.NoError(t, err)

	err = bus.Subscribe(entity3ID, func(msg Message) error {
		entity3Received <- msg
		return nil
	})
	assert.NoError(t, err)

	// Test direct messaging
	t.Run("Direct message", func(t *testing.T) {
		message := NewTextMessage(entity1ID, []string{entity2ID}, "Hello entity2")
		err = bus.Publish(message)
		assert.NoError(t, err)

		// Check that entity2 received the message
		select {
		case msg := <-entity2Received:
			content, err := msg.TextContent()
			assert.NoError(t, err)
			assert.Equal(t, "Hello entity2", content)
			assert.Equal(t, entity1ID, msg.SenderID)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for message")
		}

		// Check that entity3 did not receive the message
		select {
		case <-entity3Received:
			t.Fatal("Entity3 should not have received a message")
		case <-time.After(100 * time.Millisecond):
			// This is expected - no message should arrive
		}
	})

	// Test group messaging
	t.Run("Group message", func(t *testing.T) {
		groupID := "testGroup"

		// Create a group
		err = bus.CreateGroup(groupID, "Test Group", []string{entity2ID, entity3ID})
		assert.NoError(t, err)

		// Send a message to the group
		message := NewTextMessage(entity1ID, []string{groupID}, "Hello group")
		err = bus.Publish(message)
		assert.NoError(t, err)

		// Check that both entity2 and entity3 received the message
		select {
		case msg := <-entity2Received:
			content, err := msg.TextContent()
			assert.NoError(t, err)
			assert.Equal(t, "Hello group", content)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for message to entity2")
		}

		select {
		case msg := <-entity3Received:
			content, err := msg.TextContent()
			assert.NoError(t, err)
			assert.Equal(t, "Hello group", content)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for message to entity3")
		}
	})

	// Test broadcast messaging
	t.Run("Broadcast message", func(t *testing.T) {
		// Send a broadcast message
		message := NewTextMessage(entity1ID, []string{BroadcastAddress}, "Broadcast message")
		err = bus.Publish(message)
		assert.NoError(t, err)

		// Check that both entity2 and entity3 received the message
		select {
		case msg := <-entity2Received:
			content, err := msg.TextContent()
			assert.NoError(t, err)
			assert.Equal(t, "Broadcast message", content)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for broadcast message to entity2")
		}

		select {
		case msg := <-entity3Received:
			content, err := msg.TextContent()
			assert.NoError(t, err)
			assert.Equal(t, "Broadcast message", content)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for broadcast message to entity3")
		}

		// entity1 should not receive its own broadcast
		select {
		case <-entity1Received:
			t.Fatal("Entity1 should not receive its own broadcast")
		case <-time.After(100 * time.Millisecond):
			// This is expected behavior
		}
	})

	// Test unsubscribing
	t.Run("Unsubscribe", func(t *testing.T) {
		// Unsubscribe entity2
		err = bus.Unsubscribe(entity2ID)
		assert.NoError(t, err)

		// Send a message to entity2
		message := NewTextMessage(entity1ID, []string{entity2ID}, "This should not be received")
		err = bus.Publish(message)
		assert.NoError(t, err)

		// Check that entity2 did not receive the message
		select {
		case <-entity2Received:
			t.Fatal("Entity2 should not have received a message after unsubscribing")
		case <-time.After(100 * time.Millisecond):
			// This is expected - no message should arrive
		}
	})

	// Test group management
	t.Run("Group management", func(t *testing.T) {
		groupID := "managementGroup"

		// Create a group with entity3
		err = bus.CreateGroup(groupID, "Management Group", []string{entity3ID})
		assert.NoError(t, err)

		// Get group members
		members, err := bus.GetGroupMembers(groupID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(members))
		assert.Equal(t, entity3ID, members[0])

		// Add entity1 to the group
		err = bus.AddToGroup(groupID, entity1ID)
		assert.NoError(t, err)

		// Get updated group members
		members, err = bus.GetGroupMembers(groupID)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(members))

		// Remove entity3 from the group
		err = bus.RemoveFromGroup(groupID, entity3ID)
		assert.NoError(t, err)

		// Get updated group members
		members, err = bus.GetGroupMembers(groupID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(members))
		assert.Equal(t, entity1ID, members[0])
	})
}
