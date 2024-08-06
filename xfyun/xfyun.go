package xfyun

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/encoding/gurl"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/gogf/gf/v2/util/grand"
	"github.com/gorilla/websocket"
	"github.com/iimeta/fastapi-sdk/common"
	"github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/logger"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/fastapi-sdk/util"
	"github.com/iimeta/go-openai"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	appId               string
	secret              string
	key                 string
	originalURL         string
	baseURL             string
	path                string
	proxyURL            string
	domain              string
	isSupportSystemRole *bool
}

func NewClient(ctx context.Context, model, key, baseURL, path string, isSupportSystemRole *bool,
	endpoint string, region string, accessKey string, secretKey string,
	bucket string, domain string, proxyURL ...string) *Client {

	logger.Infof(ctx, "NewClient Xfyun model: %s, key: %s", model, key)

	result := gstr.Split(key, "|")

	client := &Client{
		appId:               result[0],
		secret:              result[1],
		key:                 result[2],
		originalURL:         "https://spark-api.xf-yun.com",
		baseURL:             "https://spark-api.xf-yun.com/v4.0",
		path:                "/chat",
		domain:              "4.0Ultra",
		isSupportSystemRole: isSupportSystemRole,
	}

	if baseURL != "" {
		logger.Infof(ctx, "NewClient Xfyun model: %s, baseURL: %s", model, baseURL)

		client.baseURL = baseURL

		version := baseURL[strings.LastIndex(baseURL, "/")+1:]

		switch version {
		case "v4.0":
			client.domain = "4.0Ultra"
		case "v3.5":
			client.domain = "generalv3.5"
		case "v3.1":
			client.domain = "generalv3"
		case "v2.1":
			client.domain = "generalv2"
		case "v1.1":
			client.domain = "general"
		default:
			v := gconv.Float64(version[1:])
			if math.Round(v) > v {
				client.domain = fmt.Sprintf("general%s", version)
			} else {
				client.domain = fmt.Sprintf("generalv%0.f", math.Round(v))
			}
		}
	}

	if path != "" {
		logger.Infof(ctx, "NewClient Xfyun model: %s, path: %s", model, path)
		client.path = path
	}

	if len(proxyURL) > 0 && proxyURL[0] != "" {
		logger.Infof(ctx, "NewClient Xfyun model: %s, proxyURL: %s", model, proxyURL[0])
		client.proxyURL = proxyURL[0]
	}

	return client
}

func (c *Client) ChatCompletion(ctx context.Context, request model.ChatCompletionRequest) (res model.ChatCompletionResponse, err error) {

	logger.Infof(ctx, "ChatCompletion Xfyun model: %s start", request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "ChatCompletion Xfyun model: %s connTime: %d ms, duration: %d ms, totalTime: %d ms", request.Model, res.ConnTime, res.Duration, res.TotalTime)
	}()

	var messages []model.ChatCompletionMessage
	if c.isSupportSystemRole != nil {
		messages = common.HandleMessages(request.Messages, *c.isSupportSystemRole)
	} else {
		messages = common.HandleMessages(request.Messages, true)
	}

	if len(messages) == 1 && messages[0].Role == consts.ROLE_SYSTEM {
		messages[0].Role = consts.ROLE_USER
	}

	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	chatCompletionReq := model.XfyunChatCompletionReq{
		Header: model.Header{
			AppId: c.appId,
			Uid:   grand.Digits(10),
		},
		Parameter: model.Parameter{
			Chat: &model.Chat{
				Domain:      c.domain,
				MaxTokens:   maxTokens,
				Temperature: request.Temperature,
				TopK:        request.N,
				ChatId:      request.User,
			},
		},
		Payload: model.Payload{
			Message: &model.Message{
				Text: messages,
			},
		},
	}

	if request.Functions != nil && len(request.Functions) > 0 {
		chatCompletionReq.Payload.Functions = new(model.Functions)
		chatCompletionReq.Payload.Functions.Text = append(chatCompletionReq.Payload.Functions.Text, request.Functions...)
	}

	data, err := gjson.Marshal(chatCompletionReq)
	if err != nil {
		logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, error: %v", request.Model, err)
		return res, err
	}

	conn, err := util.WebSocketClient(ctx, c.getWebSocketUrl(ctx), websocket.TextMessage, data, c.proxyURL)
	if err != nil {
		logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, error: %v", request.Model, err)
		return res, err
	}

	defer func() {
		if err := conn.Close(); err != nil {
			logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, conn.Close error: %v", request.Model, err)
		}
	}()

	duration := gtime.Now().UnixMilli()

	responseContent := ""
	chatCompletionRes := new(model.XfyunChatCompletionRes)

	for {

		message, err := conn.ReadMessage(ctx)
		if err != nil && !errors.Is(err, io.EOF) {
			logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, error: %v", request.Model, err)
			return res, err
		}

		if err = gjson.Unmarshal(message, &chatCompletionRes); err != nil {
			logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, message: %s, error: %v", request.Model, message, err)
			return res, errors.New(fmt.Sprintf("message: %s, error: %v", message, err))
		}

		if chatCompletionRes.Header.Code != 0 {
			logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, chatCompletionRes: %s", request.Model, gjson.MustEncodeString(chatCompletionRes))

			err = c.apiErrorHandler(chatCompletionRes)
			logger.Errorf(ctx, "ChatCompletion Xfyun model: %s, error: %v", request.Model, err)

			return res, err
		}

		responseContent += chatCompletionRes.Payload.Choices.Text[0].Content

		if chatCompletionRes.Header.Status == 2 {
			break
		}
	}

	res = model.ChatCompletionResponse{
		ID:      consts.COMPLETION_ID_PREFIX + chatCompletionRes.Header.Sid,
		Object:  consts.COMPLETION_OBJECT,
		Created: gtime.Now().Unix(),
		Model:   request.Model,
		Choices: []model.ChatCompletionChoice{{
			Index: chatCompletionRes.Payload.Choices.Seq,
			Message: &openai.ChatCompletionMessage{
				Role:         chatCompletionRes.Payload.Choices.Text[0].Role,
				Content:      responseContent,
				FunctionCall: chatCompletionRes.Payload.Choices.Text[0].FunctionCall,
			},
		}},
		Usage: &model.Usage{
			PromptTokens:     chatCompletionRes.Payload.Usage.Text.PromptTokens,
			CompletionTokens: chatCompletionRes.Payload.Usage.Text.CompletionTokens,
			TotalTokens:      chatCompletionRes.Payload.Usage.Text.TotalTokens,
		},
		ConnTime: duration - now,
		Duration: gtime.Now().UnixMilli() - duration,
	}

	return res, nil
}

func (c *Client) ChatCompletionStream(ctx context.Context, request model.ChatCompletionRequest) (responseChan chan *model.ChatCompletionResponse, err error) {

	logger.Infof(ctx, "ChatCompletionStream Xfyun model: %s start", request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		if err != nil {
			logger.Infof(ctx, "ChatCompletionStream Xfyun model: %s totalTime: %d ms", request.Model, gtime.Now().UnixMilli()-now)
		}
	}()

	var messages []model.ChatCompletionMessage
	if c.isSupportSystemRole != nil {
		messages = common.HandleMessages(request.Messages, *c.isSupportSystemRole)
	} else {
		messages = common.HandleMessages(request.Messages, true)
	}

	if len(messages) == 1 && messages[0].Role == consts.ROLE_SYSTEM {
		messages[0].Role = consts.ROLE_USER
	}

	maxTokens := request.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	chatCompletionReq := model.XfyunChatCompletionReq{
		Header: model.Header{
			AppId: c.appId,
			Uid:   grand.Digits(10),
		},
		Parameter: model.Parameter{
			Chat: &model.Chat{
				Domain:      c.domain,
				MaxTokens:   maxTokens,
				Temperature: request.Temperature,
				TopK:        request.N,
				ChatId:      request.User,
			},
		},
		Payload: model.Payload{
			Message: &model.Message{
				Text: messages,
			},
		},
	}

	if request.Functions != nil && len(request.Functions) > 0 {
		chatCompletionReq.Payload.Functions = new(model.Functions)
		chatCompletionReq.Payload.Functions.Text = append(chatCompletionReq.Payload.Functions.Text, request.Functions...)
	}

	data, err := gjson.Marshal(chatCompletionReq)
	if err != nil {
		logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, error: %v", request.Model, err)
		return responseChan, err
	}

	conn, err := util.WebSocketClient(ctx, c.getWebSocketUrl(ctx), websocket.TextMessage, data, c.proxyURL)
	if err != nil {
		logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, error: %v", request.Model, err)
		return responseChan, err
	}

	duration := gtime.Now().UnixMilli()

	responseChan = make(chan *model.ChatCompletionResponse)

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {

		defer func() {
			if err := conn.Close(); err != nil {
				logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, conn.Close error: %v", request.Model, err)
			}

			end := gtime.Now().UnixMilli()
			logger.Infof(ctx, "ChatCompletionStream Xfyun model: %s connTime: %d ms, duration: %d ms, totalTime: %d ms", request.Model, duration-now, end-duration, end-now)
		}()

		var created = gtime.Now().Unix()

		for {

			message, err := conn.ReadMessage(ctx)
			if err != nil && !errors.Is(err, io.EOF) {

				if !errors.Is(err, context.Canceled) {
					logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, error: %v", request.Model, err)
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

			chatCompletionRes := new(model.XfyunChatCompletionRes)
			if err := gjson.Unmarshal(message, &chatCompletionRes); err != nil {
				logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, message: %s, error: %v", request.Model, message, err)

				end := gtime.Now().UnixMilli()
				responseChan <- &model.ChatCompletionResponse{
					ConnTime:  duration - now,
					Duration:  end - duration,
					TotalTime: end - now,
					Error:     errors.New(fmt.Sprintf("message: %s, error: %v", message, err)),
				}

				return
			}

			if chatCompletionRes.Header.Code != 0 {
				logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, chatCompletionRes: %s", request.Model, gjson.MustEncodeString(chatCompletionRes))

				err = c.apiErrorHandler(chatCompletionRes)
				logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, error: %v", request.Model, err)

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
				ID:      consts.COMPLETION_ID_PREFIX + chatCompletionRes.Header.Sid,
				Object:  consts.COMPLETION_STREAM_OBJECT,
				Created: created,
				Model:   request.Model,
				Choices: []model.ChatCompletionChoice{{
					Index: chatCompletionRes.Payload.Choices.Seq,
					Delta: &openai.ChatCompletionStreamChoiceDelta{
						Role:         chatCompletionRes.Payload.Choices.Text[0].Role,
						Content:      chatCompletionRes.Payload.Choices.Text[0].Content,
						FunctionCall: chatCompletionRes.Payload.Choices.Text[0].FunctionCall,
					},
				}},
				ConnTime: duration - now,
			}

			if chatCompletionRes.Payload.Usage != nil {
				response.Usage = &model.Usage{
					PromptTokens:     chatCompletionRes.Payload.Usage.Text.PromptTokens,
					CompletionTokens: chatCompletionRes.Payload.Usage.Text.CompletionTokens,
					TotalTokens:      chatCompletionRes.Payload.Usage.Text.TotalTokens,
				}
			}

			if chatCompletionRes.Header.Status == 2 {

				logger.Infof(ctx, "ChatCompletionStream Xfyun model: %s finished", request.Model)

				response.Choices[0].FinishReason = openai.FinishReasonStop

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
		logger.Errorf(ctx, "ChatCompletionStream Xfyun model: %s, error: %v", request.Model, err)
		return responseChan, err
	}

	return responseChan, nil
}

func (c *Client) Image(ctx context.Context, request model.ImageRequest) (res model.ImageResponse, err error) {

	logger.Infof(ctx, "Image Xfyun model: %s start", request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "Image Xfyun model: %s totalTime: %d ms", request.Model, gtime.Now().UnixMilli()-now)
	}()

	width := 512
	height := 512

	if request.Size != "" {

		size := gstr.Split(request.Size, `×`)

		if len(size) != 2 {
			size = gstr.Split(request.Size, `x`)
		}

		if len(size) != 2 {
			size = gstr.Split(request.Size, `X`)
		}

		if len(size) != 2 {
			size = gstr.Split(request.Size, `*`)
		}

		if len(size) != 2 {
			size = gstr.Split(request.Size, `:`)
		}

		if len(size) == 2 {
			width = gconv.Int(size[0])
			height = gconv.Int(size[1])
		}
	}

	imageReq := model.XfyunChatCompletionReq{
		Header: model.Header{
			AppId: c.appId,
			Uid:   grand.Digits(10),
		},
		Parameter: model.Parameter{
			Chat: &model.Chat{
				Domain: "general",
				Width:  width,
				Height: height,
			},
		},
		Payload: model.Payload{
			Message: &model.Message{
				Text: []model.ChatCompletionMessage{{
					Role:    consts.ROLE_USER,
					Content: request.Prompt,
				}},
			},
		},
	}

	imageRes := new(model.XfyunChatCompletionRes)
	err = util.HttpPost(ctx, c.getHttpUrl(ctx), nil, imageReq, &imageRes, c.proxyURL)
	if err != nil {
		logger.Errorf(ctx, "Image Xfyun model: %s, error: %v", request.Model, err)
		return res, err
	}

	res = model.ImageResponse{
		Created: gtime.Now().Unix(),
		Data: []model.ImageResponseDataInner{{
			B64JSON: imageRes.Payload.Choices.Text[0].Content,
		}},
	}

	return res, nil
}

func (c *Client) getWebSocketUrl(ctx context.Context) string {

	date, host, signature, err := c.getSignature(ctx, http.MethodGet)
	if err != nil {
		logger.Errorf(ctx, "getWebSocketUrl Xfyun client: %+v, error: %s", c, err)
		return ""
	}

	authorizationOrigin := gbase64.EncodeToString([]byte(fmt.Sprintf("api_key=\"%s\",algorithm=\"%s\",headers=\"%s\",signature=\"%s\"", c.key, "hmac-sha256", "host date request-line", signature)))

	wsURL := gstr.Replace(gstr.Replace(c.baseURL+c.path, "https://", "wss://"), "http://", "ws://")

	return fmt.Sprintf("%s?authorization=%s&date=%s&host=%s", wsURL, authorizationOrigin, date, host)
}

func (c *Client) getHttpUrl(ctx context.Context) string {

	c.originalURL = "https://spark-api.cn-huabei-1.xf-yun.com"

	date, host, signature, err := c.getSignature(ctx, http.MethodPost)
	if err != nil {
		logger.Errorf(ctx, "getHttpUrl Xfyun client: %+v, error: %s", c, err)
		return ""
	}

	authorizationOrigin := gbase64.EncodeToString([]byte(fmt.Sprintf("api_key=\"%s\",algorithm=\"%s\",headers=\"%s\",signature=\"%s\"", c.key, "hmac-sha256", "host date request-line", signature)))

	return fmt.Sprintf("%s?authorization=%s&date=%s&host=%s", c.baseURL+c.path, authorizationOrigin, date, host)
}

func (c *Client) getSignature(ctx context.Context, method string) (date, host, signature string, err error) {

	parse, err := url.Parse(c.originalURL + c.baseURL[strings.LastIndex(c.baseURL, "/"):] + c.path)
	if err != nil {
		logger.Errorf(ctx, "getSignature Xfyun client: %+v, error: %s", c, err)
		return "", "", "", err
	}

	now := gtime.Now()
	loc, _ := time.LoadLocation("GMT")
	zone, _ := now.ToZone(loc.String())
	date = zone.Layout("Mon, 02 Jan 2006 15:04:05 GMT")

	tmp := "host: " + parse.Host + "\n"
	tmp += "date: " + date + "\n"
	tmp += method + " " + parse.Path + " HTTP/1.1"

	hash := hmac.New(sha256.New, []byte(c.secret))

	if _, err = hash.Write([]byte(tmp)); err != nil {
		logger.Errorf(ctx, "getSignature Xfyun client: %+v, error: %s", c, err)
		return "", "", "", err
	}

	return gurl.RawEncode(date), parse.Host, gbase64.EncodeToString(hash.Sum(nil)), nil
}

func (c *Client) apiErrorHandler(response *model.XfyunChatCompletionRes) error {

	switch response.Header.Code {
	case 10163, 10907:
		return sdkerr.ERR_CONTEXT_LENGTH_EXCEEDED
	}

	return sdkerr.NewApiError(500, response.Header.Code, gjson.MustEncodeString(response), "api_error", "")
}
