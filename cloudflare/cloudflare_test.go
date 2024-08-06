package cloudflare

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
	modelName := "@cf/meta/llama-3.1-8b-instruct"
	apiKey := os.Getenv("CF_API_KEY")
	accountId := os.Getenv("CF_ACCOUNT_ID")
	baseUrl := fmt.Sprintf("https://gateway.ai.cloudflare.com/v1/%s/ai_gateway/workers-ai/v1", accountId)
	path := "/chat/completions"
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

func TestImageUrl(t *testing.T) {
	endpoint := os.Getenv("endpoint")
	region := os.Getenv("region")
	accessKey := os.Getenv("accessKey")
	secretKey := os.Getenv("secretKey")
	bucket := os.Getenv("bucket")
	domain := os.Getenv("domain")
	apiKey := os.Getenv("CF_API_KEY")
	// fail if accessKey/secretKey/bucket is empty
	if accessKey == "" || secretKey == "" || bucket == "" || domain == "" {
		t.Errorf("accessKey/secretKey/bucket/domain is empty")
		t.Failed()
		return
	}

	modelName := "@cf/bytedance/stable-diffusion-xl-lightning"
	baseUrl := ""
	path := ""
	isSupportSystemRole := true
	proxyURL := ""

	ctx := context.Background()
	client := NewClient(ctx, modelName, apiKey, baseUrl, path, &isSupportSystemRole, endpoint, region, accessKey, secretKey, bucket, domain, proxyURL)
	// create an ImageRequest
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
	if imageRes.Data == nil {
		t.Errorf("未生成图片数据")
		t.Failed()
	}
	if len(imageRes.Data) != imageRequest.N {
		t.Errorf("生成图片数量不匹配")
		t.Failed()
	}
	imageData := imageRes.Data[0]
	if imageData.URL == "" {
		t.Errorf("未生成图片URL")
		t.Failed()
	}
	if imageData.B64JSON != "" {
		t.Errorf("不应该生成图片B64JSON")
		t.Failed()
	}

	fmt.Println("ImageResponse: ", imageRes.Data[0].URL)
	t.Logf("ImageResponse: %v", imageRes)
}
