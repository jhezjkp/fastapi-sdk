package compatible

// openai兼容接口

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk/common"
	"github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/logger"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/oss"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/go-openai"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	corp                string
	model               string // 实际调用的模型名称
	oss                 *oss.Oss
	apiToken            string
	baseURL             string
	path                string
	proxyURL            []string
	isSupportSystemRole *bool
}

func NewClient(ctx context.Context, corp, model, key, baseURL, path string, isSupportSystemRole *bool,
	endpoint string, region string, accessKey string, secretKey string,
	bucket string, domain string, proxyURL ...string) *Client {

	// 兼容的openai模型一定要提供baseURL
	if baseURL == "" {
		panic("baseURL is required: corp=" + corp)
	}

	return &Client{
		corp:                corp,
		model:               model,
		oss:                 &oss.Oss{Endpoint: endpoint, Region: region, AccessKey: accessKey, SecretKey: secretKey, Bucket: bucket, Domain: domain},
		apiToken:            key,
		baseURL:             baseURL,
		path:                path,
		proxyURL:            proxyURL,
		isSupportSystemRole: isSupportSystemRole,
	}
}

func (c *Client) buildOpenAiClient(ctx context.Context) *openai.Client {
	config := openai.DefaultConfig(c.apiToken)

	logger.Infof(ctx, "NewClient %s, baseURL: %s", c.corp, c.baseURL)
	config.BaseURL = c.baseURL

	if len(c.proxyURL) > 0 && c.proxyURL[0] != "" {
		logger.Infof(ctx, "NewClient %s, proxyURL: %s", c.corp, c.proxyURL[0])

		proxyUrl, err := url.Parse(c.proxyURL[0])
		if err != nil {
			panic(err)
		}

		config.HTTPClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}
	}
	return openai.NewClientWithConfig(config)
}

func (c *Client) ChatCompletion(ctx context.Context, request model.ChatCompletionRequest) (res model.ChatCompletionResponse, err error) {

	logger.Infof(ctx, "ChatCompletion %s model: %s start", c.corp, request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "ChatCompletion %s model: %s totalTime: %d ms", c.corp, request.Model, res.TotalTime)
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
			Content:      message.Content,
			FunctionCall: message.FunctionCall,
			ToolCalls:    message.ToolCalls,
			ToolCallID:   message.ToolCallID,
		}

		messages = append(messages, chatCompletionMessage)
	}

	completionRequest := openai.ChatCompletionRequest{
		Model:            request.Model,
		Messages:         messages,
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
	}
	if c.corp != consts.CORP_HYPERBOLIC && c.corp != consts.CORP_SILICONFLOW {
		completionRequest.MaxTokens = request.MaxTokens // Hyperbolic传了这个参数报400错误，先针对Hyperbolic屏蔽该参数
	}
	response, err := c.buildOpenAiClient(ctx).CreateChatCompletion(ctx, completionRequest)
	if err != nil {
		logger.Errorf(ctx, "ChatCompletion %s model: %s, error: %v", c.corp, request.Model, err)
		return res, c.apiErrorHandler(err)
	}

	logger.Infof(ctx, "ChatCompletion %s model: %s finished", c.corp, request.Model)

	res = model.ChatCompletionResponse{
		ID:      consts.COMPLETION_ID_PREFIX + response.ID,
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

	logger.Infof(ctx, "ChatCompletionStream %s model: %s start", c.corp, request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		if err != nil {
			logger.Infof(ctx, "ChatCompletionStream %s model: %s totalTime: %d ms", c.corp, request.Model, gtime.Now().UnixMilli()-now)
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
			Content:      gconv.String(message.Content),
			FunctionCall: message.FunctionCall,
			ToolCalls:    message.ToolCalls,
			ToolCallID:   message.ToolCallID,
		}

		messages = append(messages, chatCompletionMessage)
	}

	completionRequest := openai.ChatCompletionRequest{
		Model:            request.Model,
		Messages:         messages,
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
	}
	if c.corp != consts.CORP_HYPERBOLIC {
		completionRequest.MaxTokens = request.MaxTokens // Hyperbolic传了这个参数报400错误，先针对Hyperbolic屏蔽该参数
	}
	stream, err := c.buildOpenAiClient(ctx).CreateChatCompletionStream(ctx, completionRequest)
	if err != nil {
		logger.Errorf(ctx, "ChatCompletionStream %s model: %s, error: %v", c.corp, request.Model, err)
		return responseChan, c.apiErrorHandler(err)
	}

	duration := gtime.Now().UnixMilli()

	responseChan = make(chan *model.ChatCompletionResponse)

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {

		defer func() {
			if err := stream.Close(); err != nil {
				logger.Errorf(ctx, "ChatCompletionStream %s model: %s, stream.Close error: %v", c.corp, request.Model, err)
			}

			end := gtime.Now().UnixMilli()
			logger.Infof(ctx, "ChatCompletionStream %s model: %s connTime: %d ms, duration: %d ms, totalTime: %d ms", c.corp, request.Model, duration-now, end-duration, end-now)
		}()

		for {

			responseBytes, streamResponse, err := stream.Recv()
			if err != nil && !errors.Is(err, io.EOF) {

				if !errors.Is(err, context.Canceled) {
					logger.Errorf(ctx, "ChatCompletionStream %s model: %s, error: %v", c.corp, request.Model, err)
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
				ID:                consts.COMPLETION_ID_PREFIX + streamResponse.ID,
				Object:            streamResponse.Object,
				Created:           streamResponse.Created,
				Model:             streamResponse.Model,
				PromptAnnotations: streamResponse.PromptAnnotations,
				ResponseBytes:     responseBytes,
				ConnTime:          duration - now,
			}

			for _, choice := range streamResponse.Choices {
				response.Choices = append(response.Choices, model.ChatCompletionChoice{
					Index:        choice.Index,
					Delta:        &choice.Delta,
					FinishReason: choice.FinishReason,
					//ContentFilterResults: &choice.ContentFilterResults,
				})
			}

			if errors.Is(err, io.EOF) || response.Choices[0].FinishReason == openai.FinishReasonStop {
				logger.Infof(ctx, "ChatCompletionStream %s model: %s finished", c.corp, request.Model)

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
		logger.Errorf(ctx, "ChatCompletionStream %s model: %s, error: %v", c.corp, request.Model, err)
		return responseChan, err
	}

	return responseChan, nil
}

func (c *Client) apiErrorHandler(err error) error {

	apiError := &openai.APIError{}
	if errors.As(err, &apiError) {

		switch apiError.HTTPStatusCode {
		case 400:
			if apiError.Code == "context_length_exceeded" {
				return sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED
			}
		case 401:
			if apiError.Code == "invalid_api_key" {
				return sdkerr.ERR_INVALID_API_KEY
			}
		case 404:
			return sdkerr.ERR_MODEL_NOT_FOUND
		case 429:
			if apiError.Code == "insufficient_quota" {
				return sdkerr.ERR_INSUFFICIENT_QUOTA
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

func makeRequest(requestUrl string, token string, requestBody map[string]interface{}) (body io.ReadCloser, error error) {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil, err
	}
	// 创建HTTP请求
	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 发送HTTP请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return nil, err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		logger.Errorf(context.Background(), "Unexpected status code: %d, url: %v, request: %v", resp.StatusCode, requestUrl, requestBody)
		return nil, errors.New(fmt.Sprintf("unexpected status code: %d", resp.StatusCode))
	}
	return resp.Body, nil
}

func (c *Client) Embeddings(ctx context.Context, request model.EmbeddingRequest) (res model.EmbeddingResponse, err error) {
	//TODO implement me
	panic("implement me")
}

func (c *Client) Moderations(ctx context.Context, request model.ModerationRequest) (res model.ModerationResponse, err error) {
	//TODO implement me
	panic("implement me")
}
