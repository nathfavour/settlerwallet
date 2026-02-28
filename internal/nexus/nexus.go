package nexus

import (
	"context"
	"fmt"
	"sync"

	"github.com/nathfavour/settlerwallet/internal/vault"
)

// Strategy defines the interface for agentic financial logic.
type Strategy interface {
	ID() string
	Description() string
	Think(ctx context.Context, v *vault.Vault) error
}

// UserAgent represents a dedicated goroutine loop for a user.
type UserAgent struct {
	TelegramID string
	Vault      *vault.Vault
	Strategies []Strategy
	Ctx        context.Context
	Cancel     context.CancelFunc
}

// Nexus manages and dispatches user agents.
type Nexus struct {
	mu     sync.RWMutex
	agents map[string]*UserAgent
}

func NewNexus() *Nexus {
	return &Nexus{
		agents: make(map[string]*UserAgent),
	}
}

// StartUserAgent initializes and starts a goroutine for a specific user.
func (n *Nexus) StartUserAgent(telegramID string, v *vault.Vault) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.agents[telegramID]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	agent := &UserAgent{
		TelegramID: telegramID,
		Vault:      v,
		Ctx:        ctx,
		Cancel:     cancel,
	}

	n.agents[telegramID] = agent
	go n.runAgentLoop(agent)
}

func (n *Nexus) runAgentLoop(agent *UserAgent) {
	fmt.Printf("Starting agent loop for user %s
", agent.TelegramID)
	// In the future, this loop will listen for triggers and run strategies.
	for {
		select {
		case <-agent.Ctx.Done():
			fmt.Printf("Stopping agent loop for user %s
", agent.TelegramID)
			return
		}
	}
}

// StopUserAgent stops a user's goroutine.
func (n *Nexus) StopUserAgent(telegramID string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if agent, exists := n.agents[telegramID]; exists {
		agent.Cancel()
		delete(n.agents, telegramID)
	}
}
