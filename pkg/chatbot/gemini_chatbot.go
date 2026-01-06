package chatbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type GeminiChatParts struct {
	Text string `json:"text"`
}

type GeminiChatContent struct {
	Parts []*GeminiChatParts `json:"parts"`
	Role  string             `json:"role"`
}

type GeminiChatRequest struct {
	Contents         []*GeminiChatContent        `json:"contents"`
	GenerationConfig *GeminiChatGenerationConfig `json:"generationConfig"`
}

type ChatHistory struct {
	Chat string
	Role string
}

type GeminiChatCandidate struct {
	Content *GeminiChatContent `json:"content"`
}

type GeminiChatResponse struct {
	Candidates []*GeminiChatCandidate `json:"candidates"`
}

type GeminiChatPropertySchema struct {
	Type string `json:"type"`
}

type GeminiChatAppSchema struct {
	AnswerDirectly GeminiChatPropertySchema `json:"answer_directly"`
}

type GeminiChatResponseSchema struct {
	Type       string              `json:"type"`
	Properties GeminiChatAppSchema `json:"properties"`
	Required   []string            `json:"required"`
}

type GeminiChatGenerationConfig struct {
	ResponseMimeType string                    `json:"responseMimeType"`
	ResponseSchema   *GeminiChatResponseSchema `json:"responseSchema"`
}

type GeminiResponseAppSchema struct {
	AnswerDirectly bool `json:"answer_directly"`
}

// Refactored function
// GetGeminiResponse memanggil API Gemini untuk mendapatkan jawaban model
func GetGeminiResponse(ctx context.Context, apiKey string, chatHistories []*ChatHistory) (string, error) {
	var chatContents []*GeminiChatContent
	for _, chatHistory := range chatHistories {
		chatContents = append(chatContents, &GeminiChatContent{
			Parts: []*GeminiChatParts{
				{Text: chatHistory.Chat},
			},
			Role: chatHistory.Role,
		})
	}

	payload := GeminiChatRequest{
		Contents: chatContents,
		GenerationConfig: &GeminiChatGenerationConfig{
			ResponseMimeType: "text/plain", // karena kita ingin jawaban model sebagai teks
		},
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-001:generateContent",
		bytes.NewBuffer(payloadJson),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status error %d: %s", res.StatusCode, string(resBody))
	}

	// parse response Gemini
	var geminiRes GeminiChatResponse
	if err := json.Unmarshal(resBody, &geminiRes); err != nil {
		return "", err
	}

	if len(geminiRes.Candidates) == 0 || len(geminiRes.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return geminiRes.Candidates[0].Content.Parts[0].Text, nil
}

func DecideToUseRAG(
	ctx context.Context,
	apikey string,
	chatHistories []*ChatHistory,
) (bool, error) {

	var chatContents []*GeminiChatContent
	for _, chatHistory := range chatHistories {
		chatContents = append(chatContents, &GeminiChatContent{
			Parts: []*GeminiChatParts{
				{Text: chatHistory.Chat},
			},
			Role: chatHistory.Role,
		})
	}

	payload := GeminiChatRequest{
		Contents: chatContents,
		GenerationConfig: &GeminiChatGenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema: &GeminiChatResponseSchema{
				Type: "OBJECT",
				Properties: GeminiChatAppSchema{
					AnswerDirectly: GeminiChatPropertySchema{
						Type: "BOOLEAN",
					},
				},
				Required: []string{"answer_directly"},
			},
		},
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-001:generateContent",
		bytes.NewBuffer(payloadJson),
	)
	if err != nil {
		return false, err
	}

	req.Header.Set("x-goog-api-key", apikey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return false, err
	}

	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("status error %d: %s", res.StatusCode, string(resBody))
	}

	var geminiRes GeminiChatResponse
	err = json.Unmarshal(resBody, &geminiRes)
	if err != nil {
		return false, err
	}

	if len(geminiRes.Candidates) == 0 || len(geminiRes.Candidates[0].Content.Parts) == 0 {
		return false, fmt.Errorf("empty response from Gemini")
	}

	var appSchema GeminiResponseAppSchema
	err = json.Unmarshal([]byte(geminiRes.Candidates[0].Content.Parts[0].Text), &appSchema)
	if err != nil {
		return false, err
	}

	log.Printf("Use RAG: %v", !appSchema.AnswerDirectly)
	return !appSchema.AnswerDirectly, nil
}
