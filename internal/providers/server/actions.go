package server

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type actionClient struct {
	conn net.Conn
	mu   sync.Mutex
}

type Actions struct {
	mu      sync.RWMutex
	clients map[string]*actionClient
}

func NewActions() *Actions { return &Actions{clients: map[string]*actionClient{}} }

func (a *Actions) register(providerID string, conn net.Conn) func() {
	if a == nil || providerID == "" || conn == nil {
		return func() {}
	}
	client := &actionClient{conn: conn}
	a.mu.Lock()
	a.clients[providerID] = client
	a.mu.Unlock()
	return func() {
		a.mu.Lock()
		if a.clients[providerID] == client {
			delete(a.clients, providerID)
		}
		a.mu.Unlock()
	}
}

func (a *Actions) Send(providerID string, command any) error {
	if a == nil {
		return fmt.Errorf("provider action hub unavailable")
	}
	a.mu.RLock()
	client := a.clients[providerID]
	a.mu.RUnlock()
	if client == nil {
		return fmt.Errorf("provider %q is not connected", providerID)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	return json.NewEncoder(client.conn).Encode(command)
}
