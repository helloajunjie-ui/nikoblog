// Package ai 提供 OpenAI 兼容 API 的基础请求逻辑。
// 通过 Setting 中的 AiApiUrl / AiApiKey / AiModel 三个配置项发起请求。
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client OpenAI 兼容 API 客户端
type Client struct {
	APIURL string // 例如 https://api.openai.com/v1
	APIKey string
	Model  string
	HTTP   *http.Client
}

// NewClient 创建 AI 客户端。apiURL 为空时使用 OpenAI 官方地址。
func NewClient(apiURL, apiKey, model string) *Client {
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "https://api.openai.com/v1"
	}
	return &Client{
		APIURL: strings.TrimRight(strings.TrimSpace(apiURL), "/"),
		APIKey: strings.TrimSpace(apiKey),
		Model:  strings.TrimSpace(model),
		HTTP:   &http.Client{Timeout: 60 * time.Second},
	}
}

// ChatMessage 对话消息
type ChatMessage struct {
	Role    string `json:"role"` // system / user / assistant
	Content string `json:"content"`
}

// ChatCompletionRequest chat/completions 请求体
type ChatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

// ChatCompletionResponse chat/completions 响应体
type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// ChatCompletion 发起一次对话补全请求，返回助手回复内容。
func (c *Client) ChatCompletion(ctx context.Context, messages []ChatMessage) (string, error) {
	if c.Model == "" {
		return "", fmt.Errorf("AI 模型未配置")
	}
	reqBody := ChatCompletionRequest{
		Model:    c.Model,
		Messages: messages,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.APIURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 AI 服务失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI 服务返回状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed ChatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("AI 服务错误: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("AI 服务未返回内容")
	}
	return parsed.Choices[0].Message.Content, nil
}
