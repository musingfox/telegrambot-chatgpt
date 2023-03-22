package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
}

func main() {
	// 讀取 .env 檔案
	err := godotenv.Load()
	if err != nil {
		log.Fatal("無法讀取 .env 檔案")
	}

	// 讀取 BOT_TOKEN 環境變數
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN 環境變數未設定")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	// 使用 GetUpdatesChan() 取得更新
	updates := bot.GetUpdatesChan(tgbotapi.UpdateConfig{
		Timeout: 60,
	})

	var fixedMessages []ChatMessage
	var oldMessages []ChatMessage
	var newMessage ChatMessage

	// 在 fixedMessages 變數中加入固定的第一筆訊息
	fixedMessages = append(fixedMessages, ChatMessage{
		Role:    "system",
		Content: "Always response in Tranditional Chinese(zh-tw)",
	})

	for update := range updates {
		if update.Message == nil {
			continue
		}

		newMessage = ChatMessage{
			Role:    "user",
			Content: update.Message.Text,
		}

		// 檢查 oldMessages 最多只能有 8 筆訊息，這樣加上 fixedMessages 和 newMessages 最多才會有 10 筆訊息
		if len(oldMessages) > 8 {
			oldMessages = oldMessages[len(oldMessages)-8:]
		}

		// 將 fixedMessages、newMessage 和 oldMessages 合併成一個 messages 變數
		messages := append(fixedMessages, newMessage)
		if len(oldMessages) > 0 {
			messages = append(messages, oldMessages...)
		}

		reqBody, err := json.Marshal(ChatRequest{
			Model:    "gpt-3.5-turbo",
			Messages: messages,
		})
		if err != nil {
			log.Panic(err)
		}
		fmt.Printf("reqBody: %s\n", reqBody)

		req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Panic(err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Panic(err)
		}
		defer resp.Body.Close()

		// 解析 openAI API 的 response
		var chatResponse ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResponse); err != nil {
			log.Panic(err)
		}

		// 取得 usage 字串
		usageStr := fmt.Sprintf(
			"prompt_tokens: %d, completion_tokens: %d, total_tokens: %d",
			chatResponse.Usage.PromptTokens,
			chatResponse.Usage.CompletionTokens,
			chatResponse.Usage.TotalTokens,
		)

		// 將回應加上 usage 字串
		reply := chatResponse.Choices[0].Message.Content + "\n\n" + usageStr

		// 傳送回應到 Telegram
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		_, err = bot.Send(msg)
		if err != nil {
			log.Panic(err)
		}

		oldMessages = append(oldMessages, ChatMessage{
			Role:    "assistant",
			Content: chatResponse.Choices[0].Message.Content,
		})
	}
}
