package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	// 使用 GetUpdatesChan() 取得更新
	updates := bot.GetUpdatesChan(tgbotapi.UpdateConfig{
		Timeout: 60,
	})

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// 呼叫 ChatGPT API 取得回應
		resp, err := http.Get("https://api.openai.com/v1/engines/davinci-codex/completions?prompt=" + update.Message.Text + "&max_tokens=50")
		if err != nil {
			log.Panic(err)
		}

		// 讀取回應
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		// 回傳 ChatGPT 的回應
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, result["choices"].([]interface{})[0].(map[string]interface{})["text"].(string))
		bot.Send(msg)
	}
}
