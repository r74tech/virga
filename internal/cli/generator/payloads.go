package generator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// LoadPayloadSource fetches payload contents from either a local path or remote URL.
// For remote URLs (http/https) the contents are downloaded at build time to avoid
// touching the operator host once deployed.
func LoadPayloadSource(source string) ([]byte, error) {
	if source == "" {
		return nil, fmt.Errorf("payload source cannot be empty")
	}

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		resp, err := http.Get(source)
		if err != nil {
			return nil, fmt.Errorf("failed to download payload from %s: %w", source, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected HTTP status %d while downloading %s", resp.StatusCode, source)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload response from %s: %w", source, err)
		}
		return data, nil
	}

	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload file %s: %w", source, err)
	}
	return data, nil
}
