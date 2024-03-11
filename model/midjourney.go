package model

type MidjourneyProxy struct {
	ApiSecret       string `json:"api_secret"`
	ApiSecretHeader string `json:"api_secret_header"`
	ImagineUrl      string `json:"imagine_url"`
	ChangeUrl       string `json:"change_url"`
	DescribeUrl     string `json:"describe_url"`
	BlendUrl        string `json:"blend_url"`
	FetchUrl        string `json:"fetch_url"`
}

type MidjourneyProxyImagineReq struct {
	Prompt string `json:"prompt"`
	Base64 string `json:"base64"`
}
type MidjourneyProxyImagineRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
	TotalTime int64 `json:"-"`
}

type MidjourneyProxyChangeReq struct {
	Action string `json:"action"`
	Index  int    `json:"index"`
	TaskId string `json:"taskId"`
}
type MidjourneyProxyChangeRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
	TotalTime int64 `json:"-"`
}

type MidjourneyProxyDescribeReq struct {
	Base64 string `json:"base64"`
}
type MidjourneyProxyDescribeRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
	TotalTime int64 `json:"-"`
}

type MidjourneyProxyBlendReq struct {
	Base64Array []string `json:"base64Array"`
}
type MidjourneyProxyBlendRes struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	Properties  struct {
		PromptEn   string `json:"promptEn"`
		BannedWord string `json:"bannedWord"`
	} `json:"properties"`
	TotalTime int64 `json:"-"`
}

type MidjourneyProxyFetchRes struct {
	Action      string      `json:"action"`
	Id          string      `json:"id"`
	Prompt      string      `json:"prompt"`
	PromptEn    string      `json:"promptEn"`
	Description string      `json:"description"`
	State       interface{} `json:"state"`
	SubmitTime  int64       `json:"submitTime"`
	StartTime   int64       `json:"startTime"`
	FinishTime  int64       `json:"finishTime"`
	ImageUrl    string      `json:"imageUrl"`
	Status      string      `json:"status"`
	Progress    string      `json:"progress"`
	FailReason  string      `json:"failReason"`
	TotalTime   int64       `json:"-"`
}