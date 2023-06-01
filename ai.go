package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	openai "github.com/sashabaranov/go-openai"
)


func chatGPTRequest(files []string) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	c := openai.NewClient(apiKey)
	ctx := context.Background()
	prompt := fmt.Sprintf("Use the following files to tell me what languages and technologies are being used in this project: %v", files)

	req := openai.ChatCompletionRequest{
		Model:     openai.GPT3Dot5Turbo,
		MaxTokens: 150,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Stream: true,
	}
	stream, err := c.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("ChatCompletionStream error: %v\n", err)
		return
	}
	defer stream.Close()

	fmt.Printf("Stream response: ")

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("\nStream closed")
			return
		}

		if err != nil {
			fmt.Printf("\nStream error: %v\n", err)
			return
		}
		fmt.Printf(response.Choices[0].Delta.Content)
	}

}
