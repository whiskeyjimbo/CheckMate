package checkers

import (
	"fmt"
	"sync"
)

type Protocol string

type CheckerFactory func() Checker

type Registry struct {
	mu       sync.RWMutex
	checkers map[Protocol]CheckerFactory
}

var defaultRegistry = &Registry{
	checkers: make(map[Protocol]CheckerFactory),
}

func RegisterChecker(protocol Protocol, factory CheckerFactory) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.checkers[protocol] = factory
}

func NewChecker(protocol Protocol) (Checker, error) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()

	factory, exists := defaultRegistry.checkers[protocol]
	if !exists {
		return nil, fmt.Errorf("no checker registered for protocol: %s", protocol)
	}

	return factory(), nil
}

// for future use to list all protocols when some cli flag is used
func ListProtocols() []Protocol {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()

	protocols := make([]Protocol, 0, len(defaultRegistry.checkers))
	for p := range defaultRegistry.checkers {
		protocols = append(protocols, p)
	}
	return protocols
}

func (p Protocol) IsValid() bool {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	_, exists := defaultRegistry.checkers[Protocol(p)]
	return exists
}

func (p Protocol) String() string {
	return string(p)
}
