package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
)

type openai struct {
	base      string
	key       string
	client    *http.Client
	maxField  string
	listModel bool
}

func (o *openai) url(path string) string { return strings.TrimRight(o.base, "/") + path }

func (o *openai) req(ctx context.Context, method, path string, body io.Reader, ctype string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, o.url(path), body)
	if err != nil {
		return nil, err
	}
	if o.key != "" {
		req.Header.Set("Authorization", "Bearer "+o.key)
	}
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	return o.client.Do(req)
}

func (o *openai) Models(ctx context.Context) ([]ModelInfo, error) {
	resp, err := o.req(ctx, http.MethodGet, "/models", nil, "")
	if err != nil {
		return nil, &ProviderError{Message: "could not reach the provider"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, providerFail(resp)
	}
	var page struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&page); err != nil {
		return nil, err
	}
	out := make([]ModelInfo, 0, len(page.Data))
	for _, m := range page.Data {
		id := strings.TrimPrefix(m.ID, "models/")
		out = append(out, ModelInfo{ID: id, Label: id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func openaiMessages(system string, msgs []Message) []map[string]any {
	out := []map[string]any{}
	if system != "" {
		out = append(out, map[string]any{"role": "system", "content": system})
	}
	for _, m := range msgs {
		var text strings.Builder
		var calls []map[string]any
		for _, p := range m.Parts {
			switch p.Type {
			case "text":
				text.WriteString(p.Text)
			case "tool_use":
				args := string(p.Input)
				if args == "" {
					args = "{}"
				}
				calls = append(calls, map[string]any{"id": p.ID, "type": "function", "function": map[string]any{"name": p.Name, "arguments": args}})
			case "tool_result":
				out = append(out, map[string]any{"role": "tool", "tool_call_id": p.ID, "content": p.Result})
			}
		}
		if m.Role == "assistant" {
			msg := map[string]any{"role": "assistant", "content": text.String()}
			if len(calls) > 0 {
				msg["tool_calls"] = calls
				if text.Len() == 0 {
					msg["content"] = nil
				}
			}
			if text.Len() > 0 || len(calls) > 0 {
				out = append(out, msg)
			}
			continue
		}
		if text.Len() > 0 {
			out = append(out, map[string]any{"role": "user", "content": text.String()})
		}
	}
	return out
}

func (o *openai) Stream(ctx context.Context, req Request, onText func(string) error) (*Result, error) {
	body := map[string]any{"model": req.Model, "stream": true, "stream_options": map[string]any{"include_usage": true}, "messages": openaiMessages(req.System, req.Messages)}
	if req.MaxTokens > 0 {
		body[o.maxField] = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			tools = append(tools, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": t.Schema}})
		}
		body["tools"] = tools
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := o.req(ctx, http.MethodPost, "/chat/completions", bytes.NewReader(b), "application/json")
	if err != nil {
		return nil, &ProviderError{Message: "could not reach the provider"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, providerFail(resp)
	}
	res := &Result{}
	type call struct {
		id, name string
		args     strings.Builder
	}
	calls := map[int]*call{}
	var order []int
	var text strings.Builder
	err = sse(resp.Body, func(_ string, data string) error {
		if data == "[DONE]" {
			return errStop
		}
		var ch struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *struct {
				Prompt     int `json:"prompt_tokens"`
				Completion int `json:"completion_tokens"`
			} `json:"usage"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &ch) != nil {
			return nil
		}
		if ch.Error != nil {
			return &ProviderError{Message: ch.Error.Message}
		}
		if ch.Usage != nil {
			res.Usage = Usage{Input: ch.Usage.Prompt, Output: ch.Usage.Completion}
		}
		for _, c := range ch.Choices {
			if c.Delta.Content != "" {
				text.WriteString(c.Delta.Content)
				if onText != nil {
					if err := onText(c.Delta.Content); err != nil {
						return err
					}
				}
			}
			for _, tc := range c.Delta.ToolCalls {
				cur := calls[tc.Index]
				if cur == nil {
					cur = &call{}
					calls[tc.Index] = cur
					order = append(order, tc.Index)
				}
				if tc.ID != "" {
					cur.id = tc.ID
				}
				if tc.Function.Name != "" {
					cur.name = tc.Function.Name
				}
				cur.args.WriteString(tc.Function.Arguments)
			}
			if c.FinishReason != "" {
				res.Stop = c.FinishReason
			}
		}
		return nil
	})
	res.Text = text.String()
	if err != nil && err != errStop {
		return res, err
	}
	for _, i := range order {
		c := calls[i]
		input := json.RawMessage(c.args.String())
		if !json.Valid(input) {
			input = json.RawMessage("{}")
		}
		res.Calls = append(res.Calls, ToolCall{ID: c.id, Name: c.name, Input: input})
	}
	if res.Stop == "tool_calls" {
		res.Stop = "tool_use"
	}
	return res, nil
}

func (o *openai) Transcribe(ctx context.Context, model, name, mime string, audio io.Reader) (string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("model", model)
	_ = mw.WriteField("response_format", "json")
	fw, err := mw.CreateFormFile("file", name)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, io.LimitReader(audio, 25<<20)); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}
	resp, err := o.req(ctx, http.MethodPost, "/audio/transcriptions", &buf, mw.FormDataContentType())
	if err != nil {
		return "", &ProviderError{Message: "could not reach the provider"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", providerFail(resp)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&out); err != nil {
		return "", err
	}
	return out.Text, nil
}
