package protocol

// Request represents a request to the Llama integration
type Request struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data []byte `json:"data"`
}

// Response represents a response from the Llama integration
type Response struct {
	RequestID string         `json:"request_id"`
	Status    ResponseStatus `json:"status"`
	Data      []byte         `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ResponseStatus represents the status of a response
type ResponseStatus int

const (
	StatusSuccess ResponseStatus = iota
	StatusError
	StatusPending
)

// String returns the string representation of the status
func (s ResponseStatus) String() string {
	switch s {
	case StatusSuccess:
		return "success"
	case StatusError:
		return "error"
	case StatusPending:
		return "pending"
	default:
		return "unknown"
	}
}
