package compatible

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
	cases := []struct {
		Corp    string
		Model   string
		BaseURL string
		EnvKey  string
	}{
		{consts.CORP_HYPERBOLIC, "meta-llama/Meta-Llama-3.1-8B-Instruct", "https://api.hyperbolic.xyz/v1", "HYPERBOLIC_API_KEY"},
		{consts.CORP_CLOUDFLARE, "@cf/meta/llama-3.1-8b-instruct",
			fmt.Sprintf("https://gateway.ai.cloudflare.com/v1/%s/ai_gateway/workers-ai/v1",
				os.Getenv("CF_ACCOUNT_ID")), "CF_API_KEY"},
	}
	for _, c := range cases {
		t.Run(c.Corp, func(t *testing.T) {
			modelName := c.Model
			apiKey := os.Getenv(c.EnvKey)
			baseUrl := c.BaseURL
			path := ""
			isSupportSystemRole := true
			proxyURL := ""

			ctx := context.Background()
			client := NewClient(ctx, consts.CORP_HYPERBOLIC, modelName, apiKey, baseUrl, path, &isSupportSystemRole, "", "", "", "", "", "", proxyURL)
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
			content := res.Choices[0].Message.Content
			t.Logf("ChatCompletionResponse: %s", content)
			if content == "" {
				t.Errorf("ChatCompletionResponse.Messages.Content is empty")
				t.FailNow()
			}
		})
	}

}

func TestImageUrl(t *testing.T) {
	baseUrl := "https://api.siliconflow.cn/v1/%s/%s/text-to-image"
	fullUrl := fmt.Sprintf(baseUrl, "black-forest-labs", "FLUX.1-schnell")
	t.Log(fullUrl)
	if fullUrl != "https://api.siliconflow.cn/v1/black-forest-labs/FLUX.1-schnell/text-to-image" {
		t.Error("url is not correct")
		t.FailNow()
	}
}

func TestGenImageUrl(t *testing.T) {
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

	cases := []struct {
		Corp    string
		Model   string
		BaseURL string
		Path    string
		EnvKey  string
	}{
		{consts.CORP_HYPERBOLIC, "FLUX.1-dev", "https://api.hyperbolic.xyz/v1", "/image/generation", "HYPERBOLIC_API_KEY"},
		{consts.CORP_CLOUDFLARE, "@cf/bytedance/stable-diffusion-xl-lightning", "https://api.cloudflare.com/client/v4",
			fmt.Sprintf("/accounts/%s/ai/run/", os.Getenv("CF_ACCOUNT_ID")), "CF_API_KEY"},
	}
	for _, c := range cases {
		t.Run(c.Corp, func(t *testing.T) {
			corp := c.Corp
			modelName := c.Model
			apiKey := os.Getenv(c.EnvKey)
			baseUrl := c.BaseURL
			path := c.Path
			isSupportSystemRole := true
			proxyURL := ""
			ctx := context.Background()
			client := NewClient(ctx, corp, modelName, apiKey, baseUrl, path, &isSupportSystemRole, endpoint, region, accessKey, secretKey, bucket, domain, proxyURL)
			imageRequest := model.ImageRequest{
				Prompt:         "a cat sitting on a chair",
				Size:           "1024x1024",
				Model:          modelName,
				N:              1,
				ResponseFormat: openai.CreateImageResponseFormatURL,
			}
			imageRes, err := client.Image(ctx, imageRequest)
			if err != nil {
				t.Errorf("Error: %v", err)
				t.FailNow()
			}
			t.Logf("ImageResponse: %s", imageRes.Data[0].URL)
			if imageRes.Data[0].URL == "" {
				t.Errorf("ImageResponse.URL is empty")
				t.FailNow()
			}
		})
	}

}
