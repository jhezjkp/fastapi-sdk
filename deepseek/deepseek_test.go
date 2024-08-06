package deepseek

import (
	"context"
	"fmt"
	"github.com/iimeta/fastapi-sdk/model"
	"os"
	"testing"
)

func TestChatCompletion(t *testing.T) {
	supportSystemRole := true
	ctx := context.TODO()
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	c := NewClient(ctx, "deepseek-chat", apiKey, "", "", &supportSystemRole,
		"", "", "", "", "", "", "")
	if c == nil {
		t.Errorf("NewClient failed")
	}
	req := model.ChatCompletionRequest{}
	req.Model = "deepseek-chat"
	req.Messages = append(req.Messages, model.ChatCompletionMessage{Role: "user", Content: "地球直径是多少？"})
	res, err := c.ChatCompletion(ctx, req)
	if err != nil {
		t.Errorf("ChatCompletion failed, error: %v", err)
	}
	fmt.Println(res.Choices[0].Message.Content)
}
