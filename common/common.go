package common

import (
	"github.com/iimeta/fastapi-sdk/consts"
	"github.com/iimeta/fastapi-sdk/model"
	"math/rand"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandomString generates a random string of length n
func RandomString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func HandleMessages(messages []model.ChatCompletionMessage, isSupportSystemRole bool) []model.ChatCompletionMessage {

	var (
		newMessages       = make([]model.ChatCompletionMessage, 0)
		systemRoleMessage *model.ChatCompletionMessage
	)

	for _, message := range messages {
		if message.Content != "" {
			newMessages = append(newMessages, message)
		}
	}

	if isSupportSystemRole && newMessages[0].Role == consts.ROLE_SYSTEM {
		systemRoleMessage = &newMessages[0]
		newMessages = newMessages[1:]
	}

	if len(newMessages) != 0 && len(newMessages)%2 == 0 {
		newMessages = newMessages[1:]
	}

	for i := len(newMessages) - 1; i >= 0; i-- {
		if i%2 == 0 {
			newMessages[i].Role = consts.ROLE_USER
		} else {
			newMessages[i].Role = consts.ROLE_ASSISTANT
		}
	}

	if systemRoleMessage != nil {
		newMessages = append([]model.ChatCompletionMessage{*systemRoleMessage}, newMessages...)
	}

	return newMessages
}
