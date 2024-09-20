package compatible

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi-sdk/common"
	"github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/logger"
	"github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/go-openai"
	"io"
	"net/http"
)

func (c *Client) Image(ctx context.Context, request model.ImageRequest) (res model.ImageResponse, err error) {

	logger.Infof(ctx, "Image %s model: %s start", c.corp, request.Model)

	now := gtime.Now().UnixMilli()
	defer func() {
		res.TotalTime = gtime.Now().UnixMilli() - now
		logger.Infof(ctx, "Image %s model: %s totalTime: %d ms", c.corp, request.Model, gtime.Now().UnixMilli()-now)
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

	// text2image模型是特别的url
	requestUrl := fmt.Sprintf("%s%s", c.baseURL, c.path)
	if c.corp == consts.CORP_CLOUDFLARE {
		// check if requestUrl has a trailing slash
		if requestUrl[len(requestUrl)-1] != '/' {
			requestUrl += "/"
		}
		requestUrl += request.Model
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"prompt": request.Prompt,
		"width":  width,
		"height": height,
	}

	// 响应处理器: 默认为base64处理器
	bodyProcessor := func(body io.ReadCloser) (bool, string, error) {
		// convert the response body to a map
		var result map[string]interface{}
		err = json.NewDecoder(body).Decode(&result)
		if err != nil {
			logger.Errorf(context.Background(), "Error decoding response body: %v", err)
			return false, "", err
		}
		// 结果在.images[0].image，base64编码
		base64ImageData := result["images"].([]interface{})[0].(map[string]interface{})["image"].(string)
		return false, base64ImageData, nil
	}
	switch c.corp {
	case consts.CORP_HYPERBOLIC:
		requestBody["model_name"] = request.Model
	case consts.CORP_CLOUDFLARE:
		bodyProcessor = func(body io.ReadCloser) (bool, string, error) {
			// 读取响应体到byte[]
			imageBytes, err := io.ReadAll(body)
			if err != nil {
				logger.Errorf(context.Background(), "Error reading response body: %v", err)
				return false, "", err
			}
			// 返回base64编码的图片
			return false, base64.StdEncoding.EncodeToString(imageBytes), nil
		}
	case consts.CORP_SILICONFLOW: // 返回的是url
		bodyProcessor = func(body io.ReadCloser) (bool, string, error) {
			// convert the response body to a map
			var result map[string]interface{}
			err = json.NewDecoder(body).Decode(&result)
			if err != nil {
				logger.Errorf(context.Background(), "Error decoding response body: %v", err)
				return false, "", err
			}
			// 结果在.images[0].url，base64编码
			imgUrl := result["images"].([]interface{})[0].(map[string]interface{})["url"].(string)
			return true, imgUrl, nil
		}

	}

	// 构建响应
	res = model.ImageResponse{
		Created: gtime.Now().Unix(),
		// set Data to an empty list
		Data: make([]model.ImageResponseDataInner, 0),
	}

	var innerError error
	for i := 0; i < request.N; i++ {
		body, err := makeRequest(requestUrl, c.apiToken, requestBody)
		if err != nil {
			innerError = err
			continue
		}
		isImgUrl, imgResult, err := bodyProcessor(body)
		if err != nil {
			innerError = err
			continue
		}

		imageData := model.ImageResponseDataInner{}
		imageData.RevisedPrompt = request.Prompt
		switch request.ResponseFormat {
		case openai.CreateImageResponseFormatURL, "":
			var imageUrl string
			if isImgUrl {
				imageUrl = imgResult
			} else {
				// decode base64 image
				imageBytes, err := base64.StdEncoding.DecodeString(imgResult)
				if err != nil {
					logger.Errorf(ctx, "Image %s model: %s, error: %v", c.corp, request.Model, err)
					continue
				}
				logger.Infof(ctx, "image generated(size: %d), upload to oss", len(imageBytes))
				// 上传文件到云端
				imageUrl, err = c.oss.Upload(imageBytes, fmt.Sprintf("fastapi/%s.jpg", common.RandomString(8)))
				if err != nil {
					logger.Errorf(ctx, "Image %s model: %s, error: %v", c.corp, request.Model, err)
					continue
				}
			}
			imageData.URL = imageUrl
		case openai.CreateImageResponseFormatB64JSON:
			if isImgUrl {
				resp, err := http.Get(imgResult)
				if err != nil {
					logger.Errorf(ctx, "Error fetching image: %v", err)
					continue
				}
				defer resp.Body.Close()
				// Read image bytes
				imageBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					logger.Errorf(ctx, "Error reading image bytes: %v", err)
					continue
				}
				// Encode image bytes to base64
				base64Image := base64.StdEncoding.EncodeToString(imageBytes)
				imageData.B64JSON = base64Image
			} else {
				imageData.B64JSON = imgResult
			}
		default:
			return res, errors.New("invalid response format")
		}
		res.Data = append(res.Data, imageData)
	}
	// 如果N=1且有错误，则返回错误
	if request.N == 1 && innerError != nil {
		logger.Errorf(ctx, "Image %s model: %s, error: %v", c.corp, request.Model, err)
		return res, innerError
	}
	return res, nil
}
