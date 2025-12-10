package completion

// LinerAdapter adapts our Completer interface to work with liner's expectations
type LinerAdapter struct {
	completer Completer
}

// NewLinerAdapter creates a new adapter for liner
func NewLinerAdapter(completer Completer) *LinerAdapter {
	return &LinerAdapter{
		completer: completer,
	}
}

// Complete adapts our completer to liner's expected behavior
// Liner expects us to return full strings that start with the input line
func (la *LinerAdapter) Complete(line string) []string {
	if la.completer == nil {
		return nil
	}

	// Get completions from our completer
	completions, startPos := la.completer.Complete(line, len(line))

	// If no completions, return empty
	if len(completions) == 0 {
		return nil
	}

	// liner expects full strings that start with the input line
	// Our completer returns partial completions with a start position

	// Get the prefix (everything before the completion point)
	prefix := ""
	if startPos > 0 && startPos <= len(line) {
		prefix = line[:startPos]
	}

	// Build full completions
	var fullCompletions []string
	for _, comp := range completions {
		fullCompletion := prefix + comp
		// Only include completions that start with the original line
		// This handles cases where we might return multiple options
		if len(fullCompletion) >= len(line) && fullCompletion[:len(line)] == line {
			fullCompletions = append(fullCompletions, fullCompletion)
		}
	}

	return fullCompletions
}
