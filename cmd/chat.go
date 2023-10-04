package cmd

import (
	"strconv"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

func NumTokensFromMessage(prompt string, model string) (numTokens int) {
	tkm, err := tiktoken.EncodingForModel(model)
	if err != nil {
		return 0
	}

	numTokens += len(tkm.Encode(prompt, nil, nil))
	numTokens += len(tkm.Encode(openai.ChatMessageRoleUser, nil, nil))
	return numTokens
}

func MaxContentLengthExceeded(prompt string) bool {
	var (
		maxTokens int
		model     string
	)

	enableGPT4, _ := strconv.ParseBool(viper.GetString("enable_gpt_4"))

	if enableGPT4 {
		model = openai.GPT4
	} else {
		model = openai.GPT3Dot5Turbo
	}

	switch model {
	case openai.GPT3Dot5Turbo, openai.GPT3Ada:
		// maxTokens = 4096
		maxTokens = 90
	case openai.GPT4:
		maxTokens = 8192
	default:
		maxTokens = 16384
	}

	return NumTokensFromMessage(prompt, model) > maxTokens
}
