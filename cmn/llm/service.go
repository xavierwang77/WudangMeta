package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Service interface {
	Chat(msg string) (string, error)
}

type deepSeekImpl struct {
}

func NewService() Service {
	switch platform {
	case "deepseek":
		return &deepSeekImpl{}
	}

	return &deepSeekImpl{}
}

func (*deepSeekImpl) Chat(msg string) (string, error) {
	if msg == "" {
		return "", nil
	}

	// 请求消息结构
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	// 请求体结构
	type ChatRequest struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
		Stream   bool      `json:"stream"`
	}

	// 响应体结构（只展示content字段）
	type ChatResponse struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}

	url := "https://api.deepseek.com/chat/completions"

	// 构造请求体
	requestBody := ChatRequest{
		Model: deepSeekConfig.Model,
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: msg},
		},
		Stream: false,
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logger.Error("json marshal fail")
		return "", err
	}

	// 构造 HTTP 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("new request fail")
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+deepSeekConfig.ApiKey)

	// 执行请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Error("close response body fail")
		}
	}(resp.Body)

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("read response body fail")
		return "", err
	}

	// 解析响应 JSON
	var chatResp ChatResponse
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		logger.Error("json unmarshal fail")
		return "", err
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}

	logger.Warn("no response message found")
	return "", fmt.Errorf("no response message found")
}
