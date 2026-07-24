/*
 * Mythic C2 Callback Management
 * Handles callback tracking and interaction
 */

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CallbackManager manages active callbacks
type CallbackManager struct {
	client    *Client
	callbacks map[string]*Callback
	mu        sync.RWMutex
}

// NewCallbackManager creates a new callback manager
func NewCallbackManager(client *Client) *CallbackManager {
	return &CallbackManager{
		client:    client,
		callbacks: make(map[string]*Callback),
	}
}

// AddCallback adds a new callback
func (cm *CallbackManager) AddCallback(cb *Callback) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks[cb.ID] = cb
}

// RemoveCallback removes a callback
func (cm *CallbackManager) RemoveCallback(id string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.callbacks, id)
}

// GetCallback returns a callback by ID
func (cm *CallbackManager) GetCallback(id string) (*Callback, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	cb, ok := cm.callbacks[id]
	return cb, ok
}

// ListCallbacks returns all callbacks
func (cm *CallbackManager) ListCallbacks() []*Callback {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	result := make([]*Callback, 0, len(cm.callbacks))
	for _, cb := range cm.callbacks {
		result = append(result, cb)
	}
	return result
}

// RefreshCallbacks fetches callbacks from Mythic
func (cm *CallbackManager) RefreshCallbacks(ctx context.Context) error {
	result, err := cm.client.GetCallbacks(ctx, &GetCallbacksInput{})
	if err != nil {
		return err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.callbacks = make(map[string]*Callback, len(result.Callbacks))
	for _, cb := range result.Callbacks {
		cm.callbacks[cb.ID] = cb
	}

	return nil
}

// WebSocketListener listens for Mythic WebSocket events
func (cm *CallbackManager) StartListener(ctx context.Context) error {
	if !cm.client.IsConnected() {
		return fmt.Errorf("not connected to Mythic")
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var msg map[string]interface{}
				if err := cm.client.ReceiveMessage(ctx, &msg); err != nil {
					time.Sleep(5 * time.Second)
					continue
				}
				cm.handleMessage(msg)
			}
		}
	}()

	return nil
}

// handleMessage processes incoming WebSocket messages
func (cm *CallbackManager) handleMessage(msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "callback":
		cm.handleCallbackMessage(msg)
	case "task":
		cm.handleTaskMessage(msg)
	}
}

// handleCallbackMessage processes callback messages
func (cm *CallbackManager) handleCallbackMessage(msg map[string]interface{}) {
	data, _ := json.Marshal(msg["data"])
	var cb Callback
	if err := json.Unmarshal(data, &cb); err != nil {
		return
	}

	switch msg["action"] {
	case "created":
		cm.AddCallback(&cb)
	case "removed":
		cm.RemoveCallback(cb.ID)
	}
}

// handleTaskMessage processes task messages
func (cm *CallbackManager) handleTaskMessage(msg map[string]interface{}) {
	// Handle task responses
}

// CallbackStats holds callback statistics
type CallbackStats struct {
	Total    int            `json:"total"`
	ByHost   map[string]int `json:"by_host"`
	ByAgent  map[string]int `json:"by_agent"`
	ByStatus map[string]int `json:"by_status"`
}

// GetStats returns callback statistics
func (cm *CallbackManager) GetStats() *CallbackStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := &CallbackStats{
		Total:    len(cm.callbacks),
		ByHost:   make(map[string]int),
		ByAgent:  make(map[string]int),
		ByStatus: make(map[string]int),
	}

	for _, cb := range cm.callbacks {
		stats.ByHost[cb.Host]++
		stats.ByAgent[cb.AgentID]++
		stats.ByStatus[cb.Status]++
	}

	return stats
}
