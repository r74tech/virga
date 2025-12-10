//go:build !windows

package shell

import (
	"fmt"

	"github.com/r74tech/virga/internal/implant/payloads"
)

func executePowerShellPayload(payload *payloads.EmbeddedPayload, userCommand string) (string, int, error) {
	return "", 1, fmt.Errorf("powershell payloads require a Windows target")
}
