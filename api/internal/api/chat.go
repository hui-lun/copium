package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"copium/api/internal/ai"
	"copium/api/internal/store"
)

const (
	maxChatMessages = 20   // only the most recent turns are sent to the model
	maxMessageRunes = 2000 // per message
	maxContextHits  = 50   // screen hits included in the system prompt
)

// systemPrompt is in Traditional Chinese because it steers the reply language.
const systemPrompt = `你是「韭菜農場」App 裡的 AI 股市助理，服務台灣股市的散戶投資人。
- 一律使用繁體中文回答，語氣親切、簡潔、條理清楚。
- 你可以解釋選股條件、技術指標、籌碼與回測概念，並根據下方「今日選股結果」回答問題。
- 不要給出明確的買進或賣出建議，也不要保證獲利；談到個股時提醒使用者自行判斷風險。
- 沒有的資料就直說不知道，不要編造股價、財報或新聞。`

// POST /api/v1/chat  {"messages": [{"role": "user", "text": "..."}, ...]}
func (h *handlers) chat(c *gin.Context) {
	var body struct {
		Messages []ai.Message `json:"messages" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	history, err := sanitizeHistory(body.Messages)
	if err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	// Screen results are optional context; the chat still works without them.
	hits, _ := h.Store.LatestScreenResults(ctx)
	reply, err := h.AI.Chat(ctx, systemPrompt+"\n\n"+screenContext(hits), history)
	if errors.Is(err, ai.ErrDisabled) {
		fail(c, http.StatusServiceUnavailable, "AI 功能尚未設定，請在 .env 設定 GEMINI_API_KEY")
		return
	}
	if errors.Is(err, ai.ErrBusy) {
		fail(c, http.StatusServiceUnavailable, "AI 目前使用人數太多，請稍後再試")
		return
	}
	if err != nil {
		fail(c, http.StatusBadGateway, err.Error())
		return
	}
	ok(c, gin.H{"reply": reply})
}

// sanitizeHistory validates roles, trims long input and keeps the recent turns.
// Gemini expects the conversation to start with a user turn and end with one.
func sanitizeHistory(msgs []ai.Message) ([]ai.Message, error) {
	out := make([]ai.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "model" {
			return nil, fmt.Errorf("invalid role %q", m.Role)
		}
		text := strings.TrimSpace(m.Text)
		if text == "" {
			continue
		}
		if r := []rune(text); len(r) > maxMessageRunes {
			text = string(r[:maxMessageRunes])
		}
		out = append(out, ai.Message{Role: m.Role, Text: text})
	}
	if len(out) > maxChatMessages {
		out = out[len(out)-maxChatMessages:]
	}
	for len(out) > 0 && out[0].Role != "user" {
		out = out[1:]
	}
	if len(out) == 0 || out[len(out)-1].Role != "user" {
		return nil, errors.New("the last message must be from the user")
	}
	return out, nil
}

// screenContext renders today's screen hits as plain text for the model.
func screenContext(hits []store.ScreenHit) string {
	if len(hits) == 0 {
		return "今日選股結果：目前沒有符合條件的股票（或尚未執行選股）。"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "今日選股結果（%s 收盤，共 %d 檔）：\n", hits[0].Date.Format("2006-01-02"), len(hits))
	for i, h := range hits {
		if i == maxContextHits {
			fmt.Fprintf(&sb, "…其餘 %d 檔省略\n", len(hits)-maxContextHits)
			break
		}
		fmt.Fprintf(&sb, "- %s %s，收盤 %.2f，策略「%s」，符合：%s\n",
			h.Code, h.Name, h.Close, h.StrategyName, strings.Join(h.Matched, "、"))
	}
	return sb.String()
}
