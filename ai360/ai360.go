package ai360

import (
	"context"
	"errors"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk/common"
	"github.com/iimeta/fastapi-sdk/logger"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/go-openai"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	client              *openai.Client
	isSupportSystemRole *bool
}

func NewClient(ctx context.Context, model, key, baseURL, path string, isSupportSystemRole *bool,
	endpoint string, region string, accessKey string, secretKey string,
	bucket string, domain string, proxyURL ...string) *Client {

	logger.Infof(ctx, "NewClient 360AI model: %s, key: %s", model, key)

	config := openai.DefaultConfig(key)

	if baseURL != "" {
		logger.Infof(ctx, "NewClient 360AI model: %s, baseURL: %s", model, baseURL)
		config.BaseURL = baseURL
	} else {
		config.BaseURL = "https://api.360.cn/v1"
	}

	if len(proxyURL) > 0 && proxyURL[0] != "" {
		logger.Infof(ctx, "NewClient 360AI model: %s, proxyURL: %s", model, proxyURL[0])

		proxyUrl, err := url.Parse(proxyURL[0])
		if err != nil {
			panic(err)
		}

		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}

	return &Client{
		client:              openai.NewClientWithConfig(config),
		isSupportSystemRole: isSupportSystemRole,
	}
}

func (c *Client) ChatCompletion(ctx context.Context, request model.ChatCompletionRequest) (res model.ChatCompletionResponse, err error) {

	logger.Infof(ctx, "ChatCompletion 360AI model: %s start", request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "ChatCompletion 360AI model: %s totalTime: %d ms", request.Model, res.TotalTime)
	}()

	var newMessages []model.ChatCompletionMessage
	if c.isSupportSystemRole != nil {
		newMessages = common.HandleMessages(request.Messages, *c.isSupportSystemRole)
	} else {
		newMessages = common.HandleMessages(request.Messages, true)
	}

	messages := make([]openai.ChatCompletionMessage, 0)
	for _, message := range newMessages {

		chatCompletionMessage := openai.ChatCompletionMessage{
			Role:         message.Role,
			Name:         message.Name,
			FunctionCall: message.FunctionCall,
			ToolCalls:    message.ToolCalls,
			ToolCallID:   message.ToolCallID,
		}

		if content, ok := message.Content.([]interface{}); ok {
			if err = gjson.Unmarshal(gjson.MustEncode(content), &chatCompletionMessage.MultiContent); err != nil {
				return res, err
			}
		} else {
			chatCompletionMessage.Content = gconv.String(message.Content)
		}

		messages = append(messages, chatCompletionMessage)
	}

	response, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:            request.Model,
		Messages:         messages,
		MaxTokens:        request.MaxTokens,
		Temperature:      request.Temperature,
		TopP:             request.TopP,
		N:                request.N,
		Stream:           request.Stream,
		Stop:             request.Stop,
		PresencePenalty:  request.PresencePenalty,
		ResponseFormat:   request.ResponseFormat,
		Seed:             request.Seed,
		FrequencyPenalty: request.FrequencyPenalty,
		LogitBias:        request.LogitBias,
		LogProbs:         request.LogProbs,
		TopLogProbs:      request.TopLogProbs,
		User:             request.User,
		Functions:        request.Functions,
		FunctionCall:     request.FunctionCall,
		Tools:            request.Tools,
		ToolChoice:       request.ToolChoice,
	})
	if err != nil {
		logger.Errorf(ctx, "ChatCompletion 360AI model: %s, error: %v", request.Model, err)
		return res, c.apiErrorHandler(err)
	}

	logger.Infof(ctx, "ChatCompletion 360AI model: %s finished", request.Model)

	res = model.ChatCompletionResponse{
		ID:      response.ID,
		Object:  response.Object,
		Created: response.Created,
		Model:   response.Model,
		Usage: &model.Usage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
		SystemFingerprint: response.SystemFingerprint,
	}

	for _, choice := range response.Choices {
		res.Choices = append(res.Choices, model.ChatCompletionChoice{
			Index:        choice.Index,
			Message:      &choice.Message,
			FinishReason: choice.FinishReason,
			LogProbs:     choice.LogProbs,
		})
	}

	return res, nil
}

func (c *Client) ChatCompletionStream(ctx context.Context, request model.ChatCompletionRequest) (responseChan chan *model.ChatCompletionResponse, err error) {

	logger.Infof(ctx, "ChatCompletionStream 360AI model: %s start", request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		if err != nil {
			logger.Infof(ctx, "ChatCompletionStream 360AI model: %s totalTime: %d ms", request.Model, gtime.Now().UnixMilli()-now)
		}
	}()

	var newMessages []model.ChatCompletionMessage
	if c.isSupportSystemRole != nil {
		newMessages = common.HandleMessages(request.Messages, *c.isSupportSystemRole)
	} else {
		newMessages = common.HandleMessages(request.Messages, true)
	}

	messages := make([]openai.ChatCompletionMessage, 0)
	for _, message := range newMessages {

		chatCompletionMessage := openai.ChatCompletionMessage{
			Role:         message.Role,
			Name:         message.Name,
			FunctionCall: message.FunctionCall,
			ToolCalls:    message.ToolCalls,
			ToolCallID:   message.ToolCallID,
		}

		if content, ok := message.Content.([]interface{}); ok {
			if err = gjson.Unmarshal(gjson.MustEncode(content), &chatCompletionMessage.MultiContent); err != nil {
				return responseChan, err
			}
		} else {
			chatCompletionMessage.Content = gconv.String(message.Content)
		}

		messages = append(messages, chatCompletionMessage)
	}

	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:            request.Model,
		Messages:         messages,
		MaxTokens:        request.MaxTokens,
		Temperature:      request.Temperature,
		TopP:             request.TopP,
		N:                request.N,
		Stream:           request.Stream,
		Stop:             request.Stop,
		PresencePenalty:  request.PresencePenalty,
		ResponseFormat:   request.ResponseFormat,
		Seed:             request.Seed,
		FrequencyPenalty: request.FrequencyPenalty,
		LogitBias:        request.LogitBias,
		LogProbs:         request.LogProbs,
		TopLogProbs:      request.TopLogProbs,
		User:             request.User,
		Functions:        request.Functions,
		FunctionCall:     request.FunctionCall,
		Tools:            request.Tools,
		ToolChoice:       request.ToolChoice,
	})
	if err != nil {
		logger.Errorf(ctx, "ChatCompletionStream 360AI model: %s, error: %v", request.Model, err)
		return responseChan, c.apiErrorHandler(err)
	}

	duration := gtime.Now().UnixMilli()

	responseChan = make(chan *model.ChatCompletionResponse)

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {

		defer func() {
			if err := stream.Close(); err != nil {
				logger.Errorf(ctx, "ChatCompletionStream 360AI model: %s, stream.Close error: %v", request.Model, err)
			}

			end := gtime.Now().UnixMilli()
			logger.Infof(ctx, "ChatCompletionStream 360AI model: %s connTime: %d ms, duration: %d ms, totalTime: %d ms", request.Model, duration-now, end-duration, end-now)
		}()

		for {

			streamResponse, err := stream.Recv()
			if err != nil && !errors.Is(err, io.EOF) {

				if !errors.Is(err, context.Canceled) {
					logger.Errorf(ctx, "ChatCompletionStream 360AI model: %s, error: %v", request.Model, err)
				}

				end := gtime.Now().UnixMilli()
				responseChan <- &model.ChatCompletionResponse{
					ConnTime:  duration - now,
					Duration:  end - duration,
					TotalTime: end - now,
					Error:     err,
				}

				return
			}

			response := &model.ChatCompletionResponse{
				ID:                streamResponse.ID,
				Object:            streamResponse.Object,
				Created:           streamResponse.Created,
				Model:             streamResponse.Model,
				PromptAnnotations: streamResponse.PromptAnnotations,
				ConnTime:          duration - now,
			}

			for _, choice := range streamResponse.Choices {
				response.Choices = append(response.Choices, model.ChatCompletionChoice{
					Index:                choice.Index,
					Delta:                &choice.Delta,
					FinishReason:         choice.FinishReason,
					ContentFilterResults: &choice.ContentFilterResults,
				})
			}

			if streamResponse.Usage != nil {
				response.Usage = &model.Usage{
					PromptTokens:     streamResponse.Usage.PromptTokens,
					CompletionTokens: streamResponse.Usage.CompletionTokens,
					TotalTokens:      streamResponse.Usage.TotalTokens,
				}
				response.Choices[0].FinishReason = openai.FinishReasonStop
			}

			if errors.Is(err, io.EOF) || response.Choices[0].FinishReason == openai.FinishReasonStop {
				logger.Infof(ctx, "ChatCompletionStream 360AI model: %s finished", request.Model)

				if len(response.Choices) == 0 {
					response.Choices = append(response.Choices, model.ChatCompletionChoice{
						Delta:        new(openai.ChatCompletionStreamChoiceDelta),
						FinishReason: openai.FinishReasonStop,
					})
				}

				end := gtime.Now().UnixMilli()
				response.Duration = end - duration
				response.TotalTime = end - now
				responseChan <- response

				responseChan <- &model.ChatCompletionResponse{
					ConnTime:  duration - now,
					Duration:  end - duration,
					TotalTime: end - now,
					Error:     io.EOF,
				}

				return
			}

			end := gtime.Now().UnixMilli()
			response.Duration = end - duration
			response.TotalTime = end - now

			responseChan <- response
		}
	}, nil); err != nil {
		logger.Errorf(ctx, "ChatCompletionStream 360AI model: %s, error: %v", request.Model, err)
		return responseChan, err
	}

	return responseChan, nil
}

func (c *Client) Image(ctx context.Context, request model.ImageRequest) (res model.ImageResponse, err error) {

	logger.Infof(ctx, "Image 360AI model: %s start", request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "Image 360AI model: %s totalTime: %d ms", request.Model, gtime.Now().UnixMilli()-now)
	}()

	response, err := c.client.CreateImage(ctx, openai.ImageRequest{
		Prompt:         request.Prompt,
		Model:          request.Model,
		N:              request.N,
		Quality:        request.Quality,
		Size:           request.Size,
		Style:          request.Style,
		ResponseFormat: request.ResponseFormat,
		User:           request.User,
	})
	if err != nil {
		logger.Errorf(ctx, "Image 360AI model: %s, error: %v", request.Model, err)
		return res, err
	}

	data := make([]model.ImageResponseDataInner, 0)
	for _, d := range response.Data {
		data = append(data, model.ImageResponseDataInner{
			URL:           d.URL,
			B64JSON:       d.B64JSON,
			RevisedPrompt: d.RevisedPrompt,
		})
	}

	res = model.ImageResponse{
		Created: response.Created,
		Data:    data,
	}

	return res, nil
}

func (c *Client) apiErrorHandler(err error) error {

	apiError := &openai.APIError{}
	if errors.As(err, &apiError) {

		switch apiError.HTTPStatusCode {
		case 400:
			if apiError.Code == "1001" {
				return sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED
			}
		case 401:

			if apiError.Code == "1002" {
				return sdkerr.ERR_INVALID_API_KEY
			}

			if apiError.Code == "1004" || apiError.Code == "1006" {
				return sdkerr.ERR_INSUFFICIENT_QUOTA
			}

		case 404:
			return sdkerr.ERR_MODEL_NOT_FOUND
		case 429:
			if apiError.Code == "1005" {
				return sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED
			}
		}

		return err
	}

	reqError := &openai.RequestError{}
	if errors.As(err, &reqError) {
		return sdkerr.NewRequestError(apiError.HTTPStatusCode, reqError.Err)
	}

	return err
}
