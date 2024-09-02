package compatible

import (
	"context"
	"fmt"
	"github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/go-openai"
	"io"
	"os"
	"strings"
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
		{consts.CORP_DEEPSEEK, "deepseek-chat", "https://api.deepseek.com/v1", "DEEPSEEK_API_KEY"},
		{consts.CORP_SILICONFLOW, "Qwen/Qwen2-7B-Instruct", "https://api.siliconflow.cn/v1", "SILICONFLOW_API_KEY"},
		{consts.CORP_CLOUDFLARE, "@cf/meta/llama-3.1-8b-instruct",
			fmt.Sprintf("https://gateway.ai.cloudflare.com/v1/%s/ai_gateway/workers-ai/v1",
				os.Getenv("CF_ACCOUNT_ID")), "CF_API_KEY"},
	}
	for _, c := range cases {
		t.Run(c.Corp, func(t *testing.T) {
			corp := c.Corp
			modelName := c.Model
			apiKey := os.Getenv(c.EnvKey)
			baseUrl := c.BaseURL
			path := ""
			isSupportSystemRole := true
			proxyURL := ""

			ctx := context.Background()
			client := NewClient(ctx, corp, modelName, apiKey, baseUrl, path, &isSupportSystemRole, "", "", "", "", "", "", proxyURL)
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
				MaxTokens:   2048,
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
		{consts.CORP_SILICONFLOW, "FLUX.1-schnell", "https://api.siliconflow.cn/v1", "/black-forest-labs/FLUX.1-schnell/text-to-image", "SILICONFLOW_API_KEY"},
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

func TestSpeech(t *testing.T) {
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
		{consts.CORP_HYPERBOLIC, "Melo TTS", "https://api.hyperbolic.xyz/v1", "/audio/generation", "HYPERBOLIC_API_KEY"},
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
			request := model.SpeechRequest{
				Input: "a cat sitting on a chair",
				Speed: 1,
			}
			res, err := client.Speech(ctx, request)
			if err != nil {
				t.Errorf("Error: %v", err)
				t.FailNow()
			}
			audioData, _ := io.ReadAll(res)
			t.Logf("SpeechResponse length: %d", len(audioData))
			file, err := os.Create(fmt.Sprintf("%s.mp3", corp))
			if err != nil {
				panic(err) // 处理错误
			}
			defer file.Close() // 确保文件关闭

			// 将 bytes 数组写入文件
			_, err = file.Write(audioData)
			if err != nil {
				panic(err) // 处理错误
			}

			println("MP3 文件写入成功！")
		})
	}

}

func TestTranscription(t *testing.T) {
	// 注意：groq不允许中国大陆地区访问，会报403
	cases := []struct {
		Corp    string
		Model   string
		BaseURL string
		Path    string
		EnvKey  string
	}{
		{consts.CORP_GROQ, "whisper-large-v3", "https://api.groq.com/openai/v1", "/audio/transcriptions", "GROQ_API_KEY"},
		{consts.CORP_SILICONFLOW, "iic/SenseVoiceSmall", "https://api.siliconflow.cn/v1", "/audio/transcriptions", "SILICONFLOW_API_KEY"},
	}
	for _, c := range cases {
		t.Run(c.Corp, func(t *testing.T) {
			//corp := c.Corp
			modelName := c.Model
			apiKey := os.Getenv(c.EnvKey)
			baseUrl := c.BaseURL
			//path := c.Path

			// 要转录的音频文件路径
			filePath := "/Users/vivia/workspace/whisper.cpp/samples/jfk.mp3"

			// 创建一个新的 OpenAI 客户端
			config := openai.DefaultConfig(apiKey)

			config.BaseURL = baseUrl
			client := openai.NewClientWithConfig(config)

			// 打开音频文件
			audioFile, err := os.Open(filePath)
			if err != nil {
				panic(err)
			}
			defer audioFile.Close()

			// 创建音频转录请求
			req := openai.AudioRequest{
				Model:       modelName,
				FilePath:    filePath,
				Reader:      audioFile,
				Prompt:      "transcribe the following speech",
				Temperature: 0.0,
				Language:    "en",
				Format:      openai.AudioResponseFormatJSON,
			}

			// 发送请求并获取响应
			resp, err := client.CreateTranscription(context.Background(), req)
			if err != nil {
				if apiErr, ok := err.(*openai.APIError); ok {
					fmt.Printf("OpenAI API error: %v\n", apiErr)
					fmt.Printf("Response Body: %s\n", apiErr.Type)
					fmt.Printf("Request ID: %s\n", apiErr.Message)
				} else {
					fmt.Printf("Error: %v\n", err)
				}

				t.FailNow()
			}

			// 打印转录文本
			fmt.Println(resp.Text)
			// check if the transcription contains "And so my fellow Americans"
			if strings.Index(resp.Text, "And so my fellow Americans") == -1 {
				t.Errorf("Transcription does not contain expected text")
				t.FailNow()
			}
		})
	}

}
