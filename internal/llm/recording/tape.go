package recording

import (
	"encoding/json"
	"os"
	"time"

	"github.com/bakkerme/curator-ai/internal/llm"
)

// Interaction captures a single ChatCompletion request/response pair.
type Interaction struct {
	Key      string           `json:"key"`
	Request  ChatRequestJSON  `json:"request"`
	Response llm.ChatResponse `json:"response"`
	Error    string           `json:"error,omitempty"`
}

// ChatRequestJSON is a JSON-friendly representation of llm.ChatRequest.
// We use a dedicated type so that the tape file is self-contained and does not
// depend on the internal llm package's encoding choices.
type ChatRequestJSON struct {
	Model       string        `json:"model"`
	Messages    []MessageJSON `json:"messages"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// MessageJSON is a JSON-friendly representation of llm.Message.
type MessageJSON struct {
	Role    string            `json:"role"`
	Content string            `json:"content,omitempty"`
	Parts   []MessagePartJSON `json:"parts,omitempty"`
}

// MessagePartJSON is a JSON-friendly representation of llm.MessagePart.
type MessagePartJSON struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// Tape is the serialized collection of interactions.
type Tape struct {
	Interactions []Interaction `json:"interactions"`
	RecordedAt   time.Time     `json:"recorded_at"`
}

// NewTape creates an empty tape with the current timestamp.
func NewTape() *Tape {
	return &Tape{
		RecordedAt: time.Now().UTC(),
	}
}

// SaveTo writes the tape to disk as pretty-printed JSON.
func (t *Tape) SaveTo(path string) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// LoadTape reads a tape from a JSON file on disk.
func LoadTape(path string) (*Tape, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tape Tape
	if err := json.Unmarshal(data, &tape); err != nil {
		return nil, err
	}
	return &tape, nil
}

// chatRequestToJSON converts an llm.ChatRequest to the JSON-friendly form.
func chatRequestToJSON(req llm.ChatRequest) ChatRequestJSON {
	msgs := make([]MessageJSON, len(req.Messages))
	for i, m := range req.Messages {
		var parts []MessagePartJSON
		if len(m.Parts) > 0 {
			parts = make([]MessagePartJSON, len(m.Parts))
			for j, p := range m.Parts {
				parts[j] = MessagePartJSON{
					Type:     string(p.Type),
					Text:     p.Text,
					ImageURL: p.ImageURL,
				}
			}
		}
		msgs[i] = MessageJSON{
			Role:    string(m.Role),
			Content: m.Content,
			Parts:   parts,
		}
	}
	return ChatRequestJSON{
		Model:       req.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
}
