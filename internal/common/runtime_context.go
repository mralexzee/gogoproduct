package common

import (
	"goproduct/internal/knowledge"
	"goproduct/internal/messaging"
	"sync"
)

type RuntimeContext struct {
	_ops        RuntimeOptions
	_memory     knowledge.Store
	_messageBus messaging.MessageBus
	_sync       *sync.Mutex
}

type RuntimeOptions struct {
	Memory     knowledge.Store
	MessageBus messaging.MessageBus
}

func NewRuntimeContext(opt RuntimeOptions) (*RuntimeContext, error) {
	rv := new(RuntimeContext)
	rv._sync = new(sync.Mutex)
	rv._ops = opt
	rv._memory = opt.Memory

	// Initialize message bus
	if opt.MessageBus != nil {
		rv._messageBus = opt.MessageBus
	} else {
		// Create a default in-knowledge message bus if none provided
		rv._messageBus = messaging.NewMemoryMessageBus()
	}

	return rv, nil
}

func (r *RuntimeContext) GetMemory() (knowledge.Store, error) {
	r._sync.Lock()
	defer r._sync.Unlock()

	return r._memory, nil
}

func (r *RuntimeContext) SetMemory(m knowledge.Store) error {
	r._sync.Lock()
	defer r._sync.Unlock()

	r._memory = m
	return nil
}

func (r *RuntimeContext) GetMessageBus() (messaging.MessageBus, error) {
	r._sync.Lock()
	defer r._sync.Unlock()

	return r._messageBus, nil
}

func (r *RuntimeContext) SetMessageBus(m messaging.MessageBus) error {
	r._sync.Lock()
	defer r._sync.Unlock()

	r._messageBus = m
	return nil
}
