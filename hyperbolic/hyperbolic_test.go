package hyperbolic

import (
	"context"
	"fmt"
	"github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/go-openai"
	"os"
	"testing"
)

func TestChatCompletion(t *testing.T) {
	modelName := "meta-llama/Meta-Llama-3.1-8B-Instruct"
	apiKey := os.Getenv("HYPERBOLIC_API_KEY")
	baseUrl := ""
	path := ""
	isSupportSystemRole := true
	proxyURL := ""

	ctx := context.Background()
	client := NewClient(ctx, modelName, apiKey, baseUrl, path, &isSupportSystemRole, "", "", "", "", "", "", proxyURL)
	systemMsg := model.ChatCompletionMessage{
		Role:    consts.ROLE_SYSTEM,
		Content: "you are a helpful assistant.",
	}
	userMsg := model.ChatCompletionMessage{
		Role:    consts.ROLE_USER,
		Content: "月球直径多少？",
	}
	req := model.ChatCompletionRequest{
		Model: modelName,
		// create a ChatCompletionRequest with system message and user message
		Messages:    []model.ChatCompletionMessage{systemMsg, userMsg},
		MaxTokens:   8192,
		Temperature: 0.7,
		TopP:        0.9,
		N:           1,
		Stream:      false,
	}
	res, err := client.ChatCompletion(ctx, req)
	if err != nil {
		t.Errorf("Error: %v", err)
		t.Failed()
	}
	if res.Choices == nil {
		t.Errorf("ChatCompletionResponse.Messages is nil")
		t.Failed()
	}
	fmt.Println("ChatCompletionResponse: ", res.Choices[0].Message.Content)
}

func TestImage(t *testing.T) {
	endpoint := os.Getenv("endpoint")
	region := os.Getenv("region")
	accessKey := os.Getenv("accessKey")
	secretKey := os.Getenv("secretKey")
	bucket := os.Getenv("bucket")
	domain := os.Getenv("domain")
	// fail if endpoint/region/accessKey/secretKey/bucket is empty
	if endpoint == "" || region == "" || accessKey == "" || secretKey == "" || bucket == "" {
		t.Errorf("endpoint/region/accessKey/secretKey/bucket/domain is empty")
		t.Failed()
		return
	}

	modelName := "FLUX.1-dev"
	apiKey := os.Getenv("HYPERBOLIC_API_KEY")
	baseUrl := ""
	path := "/image/generation"
	isSupportSystemRole := true
	proxyURL := ""
	ctx := context.Background()
	client := NewClient(ctx, modelName, apiKey, baseUrl, path, &isSupportSystemRole, endpoint, region, accessKey, secretKey, bucket, domain, proxyURL)
	imageRequest := model.ImageRequest{
		Prompt: "A mischievous girl with pigtails, wearing a sundress, " +
			"olding a red rose with a mischievous grin, standing in front of a majestic castle, " +
			"with a playful twinkle in her eyes",
		Size:           "1024x1024",
		Model:          modelName,
		N:              1,
		ResponseFormat: openai.CreateImageResponseFormatURL,
	}
	imageRes, err := client.Image(ctx, imageRequest)
	if err != nil {
		t.Errorf("Error: %v", err)
		t.Failed()
	}
	fmt.Println("ImageResponse: ", imageRes.Data[0].URL)

}
