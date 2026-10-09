package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nodemode-ai/form.md/internal/audit"
)

type Part struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Result    string          `json:"result,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	MediaType string          `json:"media_type,omitempty"`
	Data      []byte          `json:"-"`
}

type Message struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

func Text(role, text string) Message {
	return Message{Role: role, Parts: []Part{{Type: "text", Text: text}}}
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Schema      map[string]any `json:"input_schema"`
}

type Request struct {
	Model     string
	System    string
	Messages  []Message
	Tools     []Tool
	MaxTokens int
}

type Usage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type Result struct {
	Text  string
	Calls []ToolCall
	Stop  string
	Usage Usage
}

type ModelInfo struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type Provider interface {
	Models(ctx context.Context) ([]ModelInfo, error)
	Stream(ctx context.Context, req Request, onText func(string) error) (*Result, error)
}

type Transcriber interface {
	Transcribe(ctx context.Context, model, name, mime string, audio io.Reader) (string, error)
}

type ProviderError struct {
	Status  int
	Message string
}

func (e *ProviderError) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("the AI provider answered %d: %s", e.Status, e.Message)
	}
	return "the AI provider failed: " + e.Message
}

func providerFail(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	msg := strings.TrimSpace(string(body))
	var env struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
		msg = env.Error.Message
	}
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return &ProviderError{Status: resp.StatusCode, Message: msg}
}

func sse(r io.Reader, fn func(event, data string) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	var event string
	var data bytes.Buffer
	flush := func() error {
		if data.Len() == 0 && event == "" {
			return nil
		}
		err := fn(event, data.String())
		event = ""
		data.Reset()
		return err
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if err := flush(); err != nil {
				return err
			}
		case strings.HasPrefix(line, ":"):
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(line[6:])
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(line[5:], " "))
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return flush()
}

var errStop = errors.New("stop")

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }

func auditEntry(action string) audit.Entry { return audit.Entry{Action: action} }
