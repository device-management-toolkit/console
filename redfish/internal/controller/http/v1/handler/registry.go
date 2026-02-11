// Package v1 provides Redfish message registry lookup functionality.
package v1

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrRegistryNotFound is returned when a registry is not found
	ErrRegistryNotFound = errors.New("registry not found")
	// ErrMessageNotFound is returned when a message is not found in a registry
	ErrMessageNotFound = errors.New("message not found in registry")
)

//go:embed registries/Base.1.22.0.json
var baseRegistryJSON []byte

// MessageRegistry represents a Redfish message registry
type MessageRegistry struct {
	ID              string                    `json:"Id"`
	Name            string                    `json:"Name"`
	Language        string                    `json:"Language"`
	Description     string                    `json:"Description"`
	RegistryPrefix  string                    `json:"RegistryPrefix"`
	RegistryVersion string                    `json:"RegistryVersion"`
	OwningEntity    string                    `json:"OwningEntity"`
	Messages        map[string]MessageDetails `json:"Messages"`
}

// MessageDetails contains the details of a specific message in the registry
type MessageDetails struct {
	Description     string   `json:"Description"`
	Message         string   `json:"Message"`
	MessageSeverity string   `json:"MessageSeverity"`
	NumberOfArgs    int      `json:"NumberOfArgs"`
	ParamTypes      []string `json:"ParamTypes,omitempty"`
	Resolution      string   `json:"Resolution"`
	ClearingLogic   *struct {
		ClearsIf      string   `json:"ClearsIf,omitempty"`
		ClearsAll     bool     `json:"ClearsAll,omitempty"`
		ClearsMessage []string `json:"ClearsMessage,omitempty"`
	} `json:"ClearingLogic,omitempty"`
	Deprecated string `json:"Deprecated,omitempty"`
}

// RegistryManager manages message registries
type RegistryManager struct {
	registries map[string]*MessageRegistry
	mu         sync.RWMutex
}

var (
	registryManager *RegistryManager
	once            sync.Once
)

// GetRegistryManager returns the singleton registry manager instance
func GetRegistryManager() *RegistryManager {
	once.Do(func() {
		registryManager = &RegistryManager{
			registries: make(map[string]*MessageRegistry),
		}
		// Load the Base registry
		if err := registryManager.loadBaseRegistry(); err != nil {
			// Log error but don't fail - we can still use hardcoded fallbacks
			// Note: In production, use proper logging instead of fmt.Printf
			_ = err // Registry loading errors are handled by fallback mechanisms
		}
	})

	return registryManager
}

// loadBaseRegistry loads the Base.1.22.0 registry
func (rm *RegistryManager) loadBaseRegistry() error {
	var registry MessageRegistry
	if err := json.Unmarshal(baseRegistryJSON, &registry); err != nil {
		return fmt.Errorf("failed to unmarshal Base registry: %w", err)
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.registries["Base"] = &registry

	return nil
}

// LookupMessage looks up a message from the registry
func (rm *RegistryManager) LookupMessage(registryName, messageKey string) (*RegistryMessage, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	registry, exists := rm.registries[registryName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrRegistryNotFound, registryName)
	}

	message, exists := registry.Messages[messageKey]
	if !exists {
		return nil, fmt.Errorf("%w: %s in %s", ErrMessageNotFound, messageKey, registryName)
	}

	return &RegistryMessage{
		MessageID:       fmt.Sprintf("%s.%s.%s", registry.RegistryPrefix, registry.RegistryVersion, messageKey),
		Message:         message.Message,
		Severity:        message.MessageSeverity,
		Resolution:      message.Resolution,
		RegistryPrefix:  registry.RegistryPrefix,
		RegistryVersion: registry.RegistryVersion,
		NumberOfArgs:    message.NumberOfArgs,
		ParamTypes:      message.ParamTypes,
	}, nil
}

// RegistryMessage contains the formatted message details from registry
type RegistryMessage struct {
	MessageID       string
	Message         string
	Severity        string
	Resolution      string
	RegistryPrefix  string
	RegistryVersion string
	NumberOfArgs    int
	ParamTypes      []string
}

// FormatMessage formats the message with the provided arguments
// Converts DMTF placeholders (%1, %2, etc.) to Go format specifiers (%v)
func (rm *RegistryMessage) FormatMessage(args ...interface{}) string {
	if len(args) == 0 || rm.NumberOfArgs == 0 {
		return rm.Message
	}

	// Convert DMTF placeholders (%1, %2, ...) to Go format specifiers
	message := rm.Message
	for i := 1; i <= rm.NumberOfArgs; i++ {
		placeholder := fmt.Sprintf("%%%d", i)
		message = strings.Replace(message, placeholder, "%v", 1)
	}

	return fmt.Sprintf(message, args...)
}
