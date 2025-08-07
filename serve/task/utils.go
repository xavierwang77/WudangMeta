package task

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (p LlmPrompt) ToJSONString() (string, error) {
	bytes, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func ParseLlmOutputFormatWithMarkdown(raw string) (*Fortune, error) {
	// 去掉开头和结尾的 ```json 或 ```
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
	}
	if strings.HasSuffix(raw, "```") {
		raw = strings.TrimSuffix(raw, "```")
	}

	// 再解析为结构体
	var output Fortune
	if err := json.Unmarshal([]byte(raw), &output); err != nil {
		return nil, fmt.Errorf(`failed to parse Llm output format "%s"`, raw)
	}

	return &output, nil
}
