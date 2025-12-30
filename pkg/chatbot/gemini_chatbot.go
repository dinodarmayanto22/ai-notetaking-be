package chatbot

import (
	"context"
	"net/http"
)

type GeminiChatParts struct {
}
type GeminiChatContent struct {
	Parts
}
type GeminiChatResponse struct {
	Contents
}

type ChatHistory struct {
	Chat string
	Role string
}

func GetGeminiResponse(
	ctx context.Context,
	chatHistories []*ChatHistory,
) {
	client := &http.Client{}
	payload :=
		http.NewRequest(
			"POST",
			"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-001:generateContent")

	client.Post("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-001:generateContent")

}
