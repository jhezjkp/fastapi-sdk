package compatible

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi-sdk/logger"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/go-openai"
	"io"
)

func (c *Client) Speech(ctx context.Context, request model.SpeechRequest) (res model.SpeechResponse, err error) {
	logger.Infof(ctx, "Speech %s model: %s start", c.corp, request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "Speech %s model: %s totalTime: %d ms", c.corp, request.Model, res.TotalTime)
	}()

	// 构建请求体
	requestBody := map[string]interface{}{
		"text":  request.Input,
		"speed": request.Speed,
	}

	// 响应处理器: 默认为base64处理器
	bodyProcessor := func(body io.ReadCloser) ([]byte, error) {
		defer body.Close()
		// convert the response body to a map
		var result map[string]interface{}
		err = json.NewDecoder(body).Decode(&result)
		if err != nil {
			logger.Errorf(context.Background(), "Error decoding response body: %v", err)
			return nil, err
		}
		// 结果在.audio，base64编码
		base64AudioData := result["audio"].(string)
		// base 64解码为bytes
		audioData, err := base64.StdEncoding.DecodeString(base64AudioData)
		return audioData, err
	}
	// speech模型是特别的url
	requestUrl := fmt.Sprintf("%s%s", c.baseURL, c.path)
	body, err := makeRequest(requestUrl, c.apiToken, requestBody)
	if err != nil {
		logger.Errorf(ctx, "Speech %s model: %s, error: %v", c.corp, request.Model, err)
	}
	audioData, err := bodyProcessor(body)
	if err != nil {
		logger.Errorf(ctx, "Speech %s model: %s, error: %v", c.corp, request.Model, err)
	}

	logger.Infof(ctx, "Speech %s model: %s finished", c.corp, request.Model)

	res = model.SpeechResponse{
		ReadCloser: io.NopCloser(bytes.NewReader(audioData)),
	}

	return res, nil
}

func (c *Client) Transcription(ctx context.Context, request model.AudioRequest) (res model.AudioResponse, err error) {
	logger.Infof(ctx, "Transcription %s model: %s start", c.corp, request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "Transcription %s model: %s totalTime: %d ms", c.corp, request.Model, res.TotalTime)
	}()

	response, err := c.buildOpenAiClient(ctx).CreateTranscription(ctx, openai.AudioRequest{
		Model:                  request.Model,
		FilePath:               request.FilePath,
		Reader:                 request.Reader,
		Prompt:                 request.Prompt,
		Temperature:            request.Temperature,
		Language:               request.Language,
		Format:                 request.Format,
		TimestampGranularities: request.TimestampGranularities,
	})

	if err != nil {
		logger.Errorf(ctx, "Transcription %s model: %s, error: %v", c.corp, request.Model, err)
		return res, c.apiErrorHandler(err)
	}

	logger.Infof(ctx, "Transcription %s model: %s finished", c.corp, request.Model)

	res = model.AudioResponse{
		Task:     response.Task,
		Language: response.Language,
		Duration: response.Duration,
		Segments: response.Segments,
		Words:    response.Words,
		Text:     response.Text,
	}

	return res, nil
}
