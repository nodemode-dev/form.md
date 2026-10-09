package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type anthropic struct {
	base   string
	key    string
	client *http.Client
}

func (a *anthropic) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(a.base, "/")+path, rd)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", a.key)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	return a.client.Do(req)
}

func (a *anthropic) Models(ctx context.Context) ([]ModelInfo, error) {
	var out []ModelInfo
	after := ""
	for range 10 {
		path := "/v1/models?limit=100"
		if after != "" {
			path += "&after_id=" + after
		}
		resp, err := a.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			defer resp.Body.Close()
			return nil, providerFail(resp)
		}
		var page struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, m := range page.Data {
			out = append(out, ModelInfo{ID: m.ID, Label: m.DisplayName})
		}
		if !page.HasMore || page.LastID == "" {
			break
		}
		after = page.LastID
	}
	return out, nil
}

func anthropicContent(parts []Part) []map[string]any {
	var out []map[string]any
	for _, p := range parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				out = append(out, map[string]any{"type": "text", "text": p.Text})
			}
		case "tool_use":
			input := p.Input
			if len(input) == 0 {
				input = json.RawMessage("{}")
			}
			out = append(out, map[string]any{"type": "tool_use", "id": p.ID, "name": p.Name, "input": input})
		case "tool_result":
			out = append(out, map[string]any{"type": "tool_result", "tool_use_id": p.ID, "content": p.Result, "is_error": p.IsError})
		}
	}
	return out
}

func (a *anthropic) Stream(ctx context.Context, req Request, onText func(string) error) (*Result, error) {
	msgs := []map[string]any{}
	for _, m := range req.Messages {
		role := m.Role
		if role != "assistant" {
			role = "user"
		}
		content := anthropicContent(m.Parts)
		if len(content) == 0 {
			continue
		}
		msgs = append(msgs, map[string]any{"role": role, "content": content})
	}
	body := map[string]any{"model": req.Model, "max_tokens": req.MaxTokens, "stream": true, "messages": msgs}
	if req.System != "" {
		body["system"] = req.System
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "input_schema": t.Schema})
		}
		body["tools"] = tools
	}
	resp, err := a.do(ctx, http.MethodPost, "/v1/messages", body)
	if err != nil {
		return nil, &ProviderError{Message: "could not reach the provider"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, providerFail(resp)
	}
	res := &Result{}
	type block struct {
		kind, id, name string
		json           strings.Builder
	}
	blocks := map[int]*block{}
	var text strings.Builder
	err = sse(resp.Body, func(event, data string) error {
		var ev struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"content_block"`
			Delta struct {
				Type        string `json:"type"`
				Text        string `json:"text"`
				PartialJSON string `json:"partial_json"`
				StopReason  string `json:"stop_reason"`
			} `json:"delta"`
			Message struct {
				Usage struct {
					Input int `json:"input_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage struct {
				Output int `json:"output_tokens"`
			} `json:"usage"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			return nil
		}
		switch ev.Type {
		case "message_start":
			res.Usage.Input = ev.Message.Usage.Input
		case "content_block_start":
			blocks[ev.Index] = &block{kind: ev.ContentBlock.Type, id: ev.ContentBlock.ID, name: ev.ContentBlock.Name}
		case "content_block_delta":
			b := blocks[ev.Index]
			switch ev.Delta.Type {
			case "text_delta":
				text.WriteString(ev.Delta.Text)
				if onText != nil {
					if err := onText(ev.Delta.Text); err != nil {
						return err
					}
				}
			case "input_json_delta":
				if b != nil {
					b.json.WriteString(ev.Delta.PartialJSON)
				}
			}
		case "content_block_stop":
			if b := blocks[ev.Index]; b != nil && b.kind == "tool_use" {
				input := json.RawMessage(b.json.String())
				if len(strings.TrimSpace(b.json.String())) == 0 || !json.Valid(input) {
					input = json.RawMessage("{}")
				}
				res.Calls = append(res.Calls, ToolCall{ID: b.id, Name: b.name, Input: input})
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				res.Stop = ev.Delta.StopReason
			}
			if ev.Usage.Output > 0 {
				res.Usage.Output = ev.Usage.Output
			}
		case "error":
			return &ProviderError{Message: ev.Error.Message}
		}
		return nil
	})
	res.Text = text.String()
	if err != nil {
		return res, err
	}
	return res, nil
}
