package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) telegramAPI(ctx context.Context, token, method string, payload, out any) error {
	if token == "" || strings.ContainsAny(token, "/ ?&#\n") {
		return errors.New("токен не настроен или имеет неверный формат")
	}
	var b bytes.Buffer
	if payload != nil {
		if e := json.NewEncoder(&b).Encode(payload); e != nil {
			return e
		}
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "https://api.telegram.org/bot"+token+"/"+method, &b)
	if e != nil {
		return errors.New("не удалось создать запрос")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := a.http.Do(req)
	if e != nil {
		return errors.New("Bot API недоступен или истёк таймаут")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("Bot API HTTP %d", resp.StatusCode)
	}
	var data struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if e = json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&data); e != nil || !data.OK {
		return errors.New("Bot API отклонил запрос")
	}
	if out != nil {
		return json.Unmarshal(data.Result, out)
	}
	return nil
}

type botState struct {
	Role       string `json:"role"`
	Direction  string `json:"direction"`
	LoginToken string `json:"login_token"`
	Name       string `json:"name"`
}
type button struct {
	Text   string `json:"text"`
	Data   string `json:"callback_data,omitempty"`
	URL    string `json:"url,omitempty"`
	WebApp *struct {
		URL string `json:"url"`
	} `json:"web_app,omitempty"`
}

func (a *App) sendBot(ctx context.Context, token string, chat int64, text string, rows [][]button) error {
	payload := map[string]any{"chat_id": chat, "text": text}
	if len(rows) > 0 {
		payload["reply_markup"] = map[string]any{"inline_keyboard": rows}
	}
	return a.telegramAPI(ctx, token, "sendMessage", payload, nil)
}
func (a *App) loadBotState(ctx context.Context, tg int64) (botState, error) {
	var b []byte
	var s botState
	e := a.db.QueryRow(ctx, "SELECT body FROM bot_states WHERE telegram_id=$1", tg).Scan(&b)
	if errors.Is(e, pgx.ErrNoRows) {
		return s, nil
	}
	if e != nil {
		return s, e
	}
	e = json.Unmarshal(b, &s)
	return s, e
}
func (a *App) saveBotState(ctx context.Context, tg int64, s botState) error {
	b, _ := json.Marshal(s)
	_, e := a.db.Exec(ctx, "INSERT INTO bot_states(telegram_id,body) VALUES($1,$2) ON CONFLICT(telegram_id) DO UPDATE SET body=excluded.body,updated_at=now()", tg, b)
	return e
}

type botUpdate struct {
	ID      int64 `json:"update_id"`
	Message *struct {
		Text string `json:"text"`
		Chat struct {
			ID   int64  `json:"id"`
			Type string `json:"type"`
		} `json:"chat"`
		From telegramUser `json:"from"`
	} `json:"message"`
	Callback *struct {
		ID      string       `json:"id"`
		From    telegramUser `json:"from"`
		Data    string       `json:"data"`
		Message struct {
			Chat struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"chat"`
		} `json:"message"`
	} `json:"callback_query"`
}

func (a *App) botLoop(ctx context.Context) {
	var offset int64
	lastTokenHash := ""
	for ctx.Err() == nil {
		s, token, _, e := a.settings(ctx)
		if e != nil || !s.TelegramEnabled || token == "" {
			if !pause(ctx, 5*time.Second) {
				return
			}
			continue
		}
		if hashToken(token) != lastTokenHash {
			offset = 0
			var raw string
			if a.db.QueryRow(ctx, "SELECT value FROM runtime WHERE key=$1", botOffsetKey(token)).Scan(&raw) == nil {
				offset, _ = strconv.ParseInt(raw, 10, 64)
			}
			if e = a.telegramAPI(ctx, token, "deleteWebhook", map[string]any{"drop_pending_updates": false}, nil); e != nil {
				slog.Warn("telegram webhook removal failed")
			}
			lastTokenHash = hashToken(token)
		}
		var updates []botUpdate
		e = a.telegramAPI(ctx, token, "getUpdates", map[string]any{"offset": offset, "timeout": 25, "allowed_updates": []string{"message", "callback_query"}}, &updates)
		if e != nil {
			slog.Warn("telegram poll failed", "reason", e.Error())
			if !pause(ctx, 5*time.Second) {
				return
			}
			continue
		}
		for _, up := range updates {
			for retry := 0; retry < 3; retry++ {
				e = a.handleUpdate(ctx, s, token, up)
				if e == nil {
					break
				}
				if ctx.Err() != nil {
					return
				}
				slog.Warn("telegram update retry", "update_id", up.ID, "attempt", retry+1)
				if retry < 2 && !pause(ctx, time.Duration(retry+1)*time.Second) {
					return
				}
			}
			if e != nil {
				a.audit(ctx, "telegram.update.dead_letter:"+strconv.FormatInt(up.ID, 10))
			}
			offset = up.ID + 1
			_, _ = a.db.Exec(ctx, "INSERT INTO runtime(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=excluded.value", botOffsetKey(token), strconv.FormatInt(offset, 10))
		}
	}
}
func pause(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
func (a *App) handleUpdate(ctx context.Context, settings Settings, token string, up botUpdate) error {
	var tg telegramUser
	var chat int64
	data, text := "", ""
	if up.Message != nil {
		if up.Message.Chat.Type != "private" {
			return nil
		}
		tg = up.Message.From
		chat = up.Message.Chat.ID
		text = up.Message.Text
	} else if up.Callback != nil {
		if up.Callback.Message.Chat.Type != "private" {
			return nil
		}
		tg = up.Callback.From
		chat = up.Callback.Message.Chat.ID
		data = up.Callback.Data
		_ = a.telegramAPI(ctx, token, "answerCallbackQuery", map[string]string{"callback_query_id": up.Callback.ID}, nil)
	} else {
		return nil
	}
	state, e := a.loadBotState(ctx, tg.ID)
	if e != nil {
		return e
	}
	state.Name = strings.TrimSpace(tg.FirstName + " " + tg.LastName)
	var existingID string
	if e = a.db.QueryRow(ctx, "SELECT id FROM users WHERE telegram_id=$1", tg.ID).Scan(&existingID); e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	if strings.HasPrefix(text, "/start") {
		if fields := strings.Fields(text); len(fields) > 1 && strings.HasPrefix(fields[1], "login_") {
			candidate := strings.TrimPrefix(fields[1], "login_")
			var valid bool
			if e = a.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM login_links WHERE token_hash=$1 AND expires_at>now() AND NOT consumed AND user_id IS NULL)", hashToken(candidate)).Scan(&valid); e != nil {
				return e
			}
			if !valid {
				return a.sendBot(ctx, token, chat, "Ссылка входа истекла. Создайте новую на сайте.", nil)
			}
			state.LoginToken = candidate
		}
		if e := a.saveBotState(ctx, tg.ID, state); e != nil {
			return e
		}
		if existingID != "" {
			return a.botReady(ctx, settings, token, chat, state)
		}
		return a.sendBot(ctx, token, chat, "Привет! Я AI-buddy — твой помощник в адаптации. Помогу разобраться в рабочих инструментах, пройти обучение и подготовиться к встречам. Это учебная демонстрация с синтетическими данными. Выбери роль:", [][]button{{{Text: "Я стажёр", Data: "role:intern"}}, {{Text: "Я наставник", Data: "role:mentor"}}, {{Text: "Я руководитель", Data: "role:manager"}}})
	}
	if data == "confirm_login" {
		if existingID == "" || state.LoginToken == "" {
			return a.sendBot(ctx, token, chat, "Начните вход заново на сайте.", nil)
		}
		res, e := a.db.Exec(ctx, "UPDATE login_links SET user_id=$2 WHERE token_hash=$1 AND user_id IS NULL AND expires_at>now() AND NOT consumed", hashToken(state.LoginToken), existingID)
		if e != nil {
			return e
		}
		state.LoginToken = ""
		_ = a.saveBotState(ctx, tg.ID, state)
		if res.RowsAffected() == 0 {
			return a.sendBot(ctx, token, chat, "Ссылка уже использована или истекла.", nil)
		}
		return a.sendBot(ctx, token, chat, "Вход подтверждён. Вернитесь в тот браузер, где запросили вход.", nil)
	}
	if existingID != "" {
		return a.botReady(ctx, settings, token, chat, state)
	}
	if strings.HasPrefix(data, "role:") {
		role := strings.TrimPrefix(data, "role:")
		if role != "intern" && role != "mentor" && role != "manager" {
			return errors.New("invalid role")
		}
		state.Role = role
		state.Direction = ""
		if e := a.saveBotState(ctx, tg.ID, state); e != nil {
			return e
		}
		rows := [][]button{}
		for _, t := range tracks() {
			rows = append(rows, []button{{Text: t.Name, Data: "direction:" + t.ID}})
		}
		return a.sendBot(ctx, token, chat, "Выберите направление:", rows)
	}
	if strings.HasPrefix(data, "direction:") && state.Role != "" {
		dir := strings.TrimPrefix(data, "direction:")
		state.Direction = dir
		teams, e := a.teams(ctx)
		if e != nil {
			return e
		}
		rows := [][]button{}
		for _, t := range teams {
			if t.Direction == dir {
				rows = append(rows, []button{{Text: t.Name, Data: "team:" + t.ID}})
			}
		}
		if len(rows) == 0 {
			return errors.New("invalid direction")
		}
		if e = a.saveBotState(ctx, tg.ID, state); e != nil {
			return e
		}
		return a.sendBot(ctx, token, chat, "Выберите учебную команду:", rows)
	}
	if strings.HasPrefix(data, "team:") && state.Role != "" && state.Direction != "" {
		team := strings.TrimPrefix(data, "team:")
		var direction string
		if e := a.db.QueryRow(ctx, "SELECT direction FROM teams WHERE id=$1", team).Scan(&direction); e != nil || direction != state.Direction {
			return errors.New("invalid team")
		}
		u, e := a.createUser(ctx, state.Name, state.Role, team, tg.ID, false)
		if e != nil {
			return e
		}
		if u.Role == "mentor" {
			_, _ = a.db.Exec(ctx, "UPDATE users SET mentor_id=$1 WHERE team_id=$2 AND role='intern' AND mentor_id IS NULL AND NOT is_demo", u.ID, u.TeamID)
		}
		return a.botReady(ctx, settings, token, chat, state)
	}
	return a.sendBot(ctx, token, chat, "Используйте /start для регистрации и открытия AI-buddy.", nil)
}
func (a *App) botReady(ctx context.Context, s Settings, token string, chat int64, state botState) error {
	rows := [][]button{}
	if state.LoginToken != "" {
		rows = append(rows, []button{{Text: "Подтвердить вход в браузере", Data: "confirm_login"}})
	}
	if strings.HasPrefix(s.AppURL, "https://") {
		rows = append(rows, []button{{Text: "Открыть AI-buddy", WebApp: &struct {
			URL string `json:"url"`
		}{s.AppURL}}})
	} else if s.AppURL != "" {
		rows = append(rows, []button{{Text: "Открыть AI-buddy", URL: s.AppURL}})
	}
	return a.sendBot(ctx, token, chat, "Готово! Ваш профиль создан. Можно открыть приложение. Подтверждайте вход только если сами запросили его в браузере.", rows)
}
func (a *App) scheduler(ctx context.Context) {
	_, _ = a.db.Exec(ctx, `UPDATE records SET body=jsonb_set(body,'{notification_status}','"retry"') WHERE kind='meeting' AND body->>'notification_status'='sending'`)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		a.sendReminders(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (a *App) sendReminders(ctx context.Context) {
	s, token, _, e := a.settings(ctx)
	if e != nil {
		return
	}
	rows, e := a.db.Query(ctx, `SELECT r.id,r.user_id FROM records r JOIN users u ON u.id=r.user_id WHERE r.kind='meeting' AND u.notifications AND (r.body->>'notification_status') IN ('pending','retry','waiting_configuration') AND (r.body->>'starts_at')::timestamptz>now()-interval '1 hour' AND (r.body->>'starts_at')::timestamptz <= now()+make_interval(mins=>u.reminder_minutes) AND (r.body->>'retry_at' IS NULL OR (r.body->>'retry_at')::timestamptz<now()) LIMIT 100`)
	if e != nil {
		return
	}
	type item struct{ key, owner string }
	items := []item{}
	for rows.Next() {
		var x item
		if rows.Scan(&x.key, &x.owner) == nil {
			items = append(items, x)
		}
	}
	rows.Close()
	for _, x := range items {
		u, e := a.loadUser(ctx, x.owner)
		if e != nil {
			continue
		}
		m, _, e := record[Meeting](ctx, a, "meeting", x.key)
		if e != nil {
			continue
		}
		if u.TelegramID == 0 {
			m.NotificationStatus = "demo_reminder"
			_ = a.save(ctx, "meeting", u.ID, m.ID, m)
			continue
		}
		if !s.TelegramEnabled || token == "" {
			_, _ = a.db.Exec(ctx, `UPDATE records SET body=jsonb_set(jsonb_set(body,'{notification_status}','"waiting_configuration"'),'{retry_at}',to_jsonb((now()+interval '1 minute')::text)) WHERE id=$1`, m.ID)
			continue
		}
		res, e := a.db.Exec(ctx, `UPDATE records SET body=jsonb_set(body,'{notification_status}','"sending"') WHERE id=$1 AND body->>'notification_status' IN ('pending','retry','waiting_configuration')`, m.ID)
		if e != nil || res.RowsAffected() == 0 {
			continue
		}
		url := s.AppURL + "/knowledge/" + m.ArticleID
		e = a.sendBot(ctx, token, u.TelegramID, "Скоро встреча «"+m.Title+"» ("+m.StartsAt.In(time.Local).Format("02.01 15:04")+"). Подготовьтесь по учебному материалу.", [][]button{{{Text: "Изучить материал", URL: url}}})
		if e == nil {
			m.NotificationStatus = "sent"
			_ = a.save(ctx, "meeting", u.ID, m.ID, m)
		} else {
			_, _ = a.db.Exec(ctx, `UPDATE records SET body=jsonb_set(jsonb_set(body,'{notification_status}','"retry"'),'{retry_at}',to_jsonb((now()+interval '5 minutes')::text)) WHERE id=$1`, m.ID)
		}
	}
	_, _ = a.db.Exec(ctx, "DELETE FROM sessions WHERE expires_at<now(); DELETE FROM login_links WHERE expires_at<now(); DELETE FROM bot_states WHERE updated_at<now()-interval '7 days'")
}

func botOffsetKey(token string) string { return "telegram_offset_" + hashToken(token) }
