package llm

import "testing"

func TestChat(t *testing.T) {
	Init()

	service := NewService()

	msg := "Hello, how are you?"

	response, err := service.Chat(msg)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	t.Log(response)
}
