package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type apiClient struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
}

func (c *apiClient) call(method, path string, data any, want int) []byte {
	c.t.Helper()
	var b bytes.Buffer
	if data != nil {
		json.NewEncoder(&b).Encode(data)
	}
	r := httptest.NewRequest(method, path, &b)
	r.Header.Set("Content-Type", "application/json")
	if c.cookie != nil {
		r.AddCookie(c.cookie)
	}
	w := httptest.NewRecorder()
	c.h.ServeHTTP(w, r)
	if w.Code != want {
		c.t.Fatalf("%s %s: wanted %d got %d: %s", method, path, want, w.Code, w.Body.String())
	}
	if len(w.Result().Cookies()) > 0 {
		c.cookie = w.Result().Cookies()[0]
	}
	return w.Body.Bytes()
}
func testApp(t *testing.T) *App {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL tests")
	}
	a, e := newApp(context.Background(), url, "test-password-very-long", "test-encryption-key-over-32-characters", "http://example.test", false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(a.db.Close)
	return a
}
func demo(t *testing.T, a *App, role, team string) (*apiClient, User) {
	t.Helper()
	c := &apiClient{t: t, h: a.routes()}
	b := c.call("POST", "/api/auth/demo", map[string]string{"role": role, "team_id": team}, 200)
	var u User
	json.Unmarshal(b, &u)
	return c, u
}
func TestAPIScenarios(t *testing.T) {
	a := testApp(t)
	c, u := demo(t, a, "intern", "")
	defer a.db.Exec(context.Background(), "DELETE FROM users WHERE id=$1", u.ID)
	c.call("GET", "/health", nil, 200)
	c.call("GET", "/api/public/config", nil, 200)
	c.call("GET", "/api/teams", nil, 200)
	c.call("PATCH", "/api/me", map[string]any{"notifications": false, "reminder_minutes": 30}, 200)
	c.call("PATCH", "/api/me", map[string]any{"reminder_minutes": 0}, 400)
	var tasks []Task
	json.Unmarshal(c.call("GET", "/api/tasks", nil, 200), &tasks)
	for _, task := range tasks {
		if task.Kind == "manual" {
			c.call("PATCH", "/api/tasks/"+task.ID, map[string]string{"status": "in_progress"}, 200)
			c.call("PATCH", "/api/tasks/"+task.ID, map[string]string{"status": "completed"}, 200)
			c.call("PATCH", "/api/tasks/"+task.ID, map[string]string{"status": "madeup"}, 400)
			break
		}
	}
	c.call("GET", "/api/knowledge?q=Git&direction=backend", nil, 200)
	c.call("GET", "/api/knowledge/track-backend", nil, 200)
	c.call("POST", "/api/knowledge/track-backend/read", map[string]any{}, 200)
	b := c.call("GET", "/api/knowledge/track-backend", nil, 200)
	var ar Article
	json.Unmarshal(b, &ar)
	if !ar.Read {
		t.Fatal("read mark did not persist")
	}
	c.call("GET", "/api/knowledge/no-such-article", nil, 404)
	c.call("GET", "/api/lessons", nil, 200)
	c.call("POST", "/api/lessons/lesson-backend/exam", map[string]any{"answers": map[string]int{}}, 400)
	c.call("POST", "/api/chat", map[string]string{"message": "Как получить доступ к GitLab?"}, 200)
	c.call("POST", "/api/chat", map[string]string{"message": "Объясни теорию", "lesson_id": "lesson-backend"}, 200)
	c.call("GET", "/api/chat", nil, 200)
	c.call("POST", "/api/chat", map[string]string{"message": ""}, 400)
	c.call("POST", "/api/chat", map[string]string{"message": strings.Repeat("Объясни материал ", 30)}, 200)
	c.call("POST", "/api/requests", map[string]any{"resource": "GitLab", "reason": "test", "confirmed": false}, 400)
	c.call("GET", "/api/requests", nil, 200)
	c.call("GET", "/api/meetings", nil, 200)
	c.call("POST", "/api/meetings/demo", map[string]any{}, 200)
	c.call("GET", "/api/dashboard", nil, 200)
	mentor, _ := demo(t, a, "mentor", u.TeamID)
	mentor.call("GET", "/api/interns", nil, 200)
	mentor.call("GET", "/api/dashboard", nil, 200)
	mentor.call("POST", "/api/interns/"+u.ID+"/tasks", map[string]any{"title": "Обсудить решение", "description": "Проверка", "due_at": time.Now().Add(24 * time.Hour), "article_id": "track-backend"}, 200)
	mentor.call("POST", "/api/interns/"+u.ID+"/meetings", map[string]any{"title": "Разбор API", "starts_at": time.Now().Add(time.Hour), "article_id": "track-backend"}, 200)
	mentor.call("GET", "/api/interns/"+u.ID, nil, 200)
	manager, mu := demo(t, a, "manager", u.TeamID)
	defer a.db.Exec(context.Background(), "DELETE FROM users WHERE id=$1", mu.ID)
	manager.call("GET", "/api/dashboard", nil, 200)
	manager.call("GET", "/api/requests", nil, 200)
	manager.call("POST", "/api/interns/"+u.ID+"/tasks", map[string]any{}, 403)
	other, ou := demo(t, a, "intern", "")
	defer a.db.Exec(context.Background(), "DELETE FROM users WHERE id=$1", ou.ID)
	other.call("PATCH", "/api/tasks/"+tasks[0].ID, map[string]string{"status": "completed"}, 403)
	// Distinct auth and secret storage/masking.
	admin := &apiClient{t: t, h: a.routes()}
	admin.call("POST", "/api/admin/login", map[string]string{"password": "wrong"}, 401)
	admin.call("POST", "/api/admin/login", map[string]string{"password": "test-password-very-long"}, 200)
	var settings Settings
	json.Unmarshal(admin.call("GET", "/api/admin/settings", nil, 200), &settings)
	original := settings
	settings.RouterAIBaseURL = "http://127.0.0.1"
	admin.call("PUT", "/api/admin/settings", settings, 400)
	settings = original
	update := map[string]any{}
	b, _ = json.Marshal(settings)
	json.Unmarshal(b, &update)
	update["routerai_api_key"] = "secret-routerai-testing-key"
	admin.call("PUT", "/api/admin/settings", update, 200)
	b = admin.call("GET", "/api/admin/settings", nil, 200)
	if strings.Contains(string(b), "secret-routerai") {
		t.Fatal("secret returned")
	}
	var stored string
	a.db.QueryRow(context.Background(), "SELECT routerai_secret FROM settings WHERE id=1").Scan(&stored)
	if stored == "secret-routerai-testing-key" {
		t.Fatal("secret stored plaintext")
	}
	update["routerai_api_key"] = ""
	admin.call("PUT", "/api/admin/settings", update, 200)
	_, _, key, e := a.settings(context.Background())
	if e != nil || key != "secret-routerai-testing-key" {
		t.Fatal("blank did not preserve key", e)
	}
	update["clear_routerai_key"] = true
	admin.call("PUT", "/api/admin/settings", update, 200)
	admin.call("POST", "/api/admin/test/routerai", map[string]any{}, 200)
	admin.call("POST", "/api/admin/test/telegram", map[string]any{}, 200)
	admin.call("GET", "/api/admin/audit", nil, 200)
	admin.call("POST", "/api/admin/logout", map[string]any{}, 200)
	admin.call("GET", "/api/admin/settings", nil, 401)
	c.call("POST", "/api/auth/logout", map[string]any{}, 200)
	c.call("GET", "/api/me", nil, 401)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fakeHTTP(t *testing.T, fn func(*http.Request) string) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(fn(r))), Header: http.Header{}}, nil
	})}
}
func TestLiveProviderToolsAndTelegram(t *testing.T) {
	a := testApp(t)
	c, u := demo(t, a, "intern", "")
	ctx := context.Background()
	defer a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID)
	s, _, _, _ := a.settings(ctx)
	original := s
	s.AIMockMode = false
	s.RouterAIModel = "test/tool-model"
	s.TelegramUsername = "buddy_test_bot"
	s.TelegramEnabled = true
	te, _ := a.cipher.seal("12345:test-token")
	ke, _ := a.cipher.seal("test-routerai-key")
	b, _ := json.Marshal(s)
	_, e := a.db.Exec(ctx, "UPDATE settings SET body=$1,telegram_secret=$2,routerai_secret=$3 WHERE id=1", b, te, ke)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		b, _ := json.Marshal(original)
		a.db.Exec(ctx, "UPDATE settings SET body=$1,telegram_secret='',routerai_secret='' WHERE id=1", b)
	}()
	calls := 0
	a.http = fakeHTTP(t, func(r *http.Request) string {
		if r.URL.Host == "api.telegram.org" {
			if strings.HasSuffix(r.URL.Path, "getMe") {
				return `{"ok":true,"result":{"username":"buddy_test_bot"}}`
			}
			return `{"ok":true,"result":{}}`
		}
		if r.Header.Get("Authorization") != "Bearer test-routerai-key" {
			t.Error("missing provider key")
		}
		if strings.HasSuffix(r.URL.Path, "/models") {
			return `{"data":[{"id":"test/tool-model"}]}`
		}
		raw, _ := io.ReadAll(r.Body)
		if strings.Contains(string(raw), `"tool_choice"`) {
			return `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"tool-test","type":"function","function":{"name":"get_user_profile","arguments":"{}"}}]}}]}`
		}
		calls++
		if calls == 1 {
			return `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"tc1","type":"function","function":{"name":"search_knowledge_base","arguments":"{\"query\":\"Git\"}"}},{"id":"tc2","type":"function","function":{"name":"create_access_request","arguments":"{\"resource_id\":\"GitLab\",\"reason\":\"Учебная задача\"}"}}]}}]}`
		}
		return `{"choices":[{"message":{"role":"assistant","content":"Изучите инструкцию и подтвердите заявку в форме."}}]}`
	})
	var msg Message
	json.Unmarshal(c.call("POST", "/api/chat", map[string]string{"message": "Нужен доступ к GitLab"}, 200), &msg)
	if len(msg.Sources) == 0 || len(msg.SuggestedActions) == 0 {
		t.Fatal("missing tool sources/confirmation action", msg)
	}
	reqs, _ := records[Request](ctx, a, "request", u.ID)
	if len(reqs) != 0 {
		t.Fatal("AI executed mutation without user confirmation")
	}
	for _, name := range []string{"get_user_profile", "get_onboarding_progress", "get_upcoming_meetings", "get_learning_recommendations", "get_access_request_status", "unknown"} {
		call := aiToolCall{}
		call.Function.Name = name
		call.Function.Arguments = `{}`
		a.executeTool(ctx, u, call)
	}
	admin := &apiClient{t: t, h: a.routes()}
	admin.call("POST", "/api/admin/login", map[string]string{"password": "test-password-very-long"}, 200)
	admin.call("POST", "/api/admin/test/routerai", map[string]any{}, 200)
	admin.call("POST", "/api/admin/test/telegram", map[string]any{}, 200)
	// Complete registration using actual persisted bot state and API messages.
	tgID := time.Now().UnixNano()%1000000000 + 10000
	tg := telegramUser{ID: tgID, FirstName: "Тест Telegram"}
	send := func(text, data string) {
		var up botUpdate
		if data == "" {
			up.Message = &struct {
				Text string `json:"text"`
				Chat struct {
					ID   int64  `json:"id"`
					Type string `json:"type"`
				} `json:"chat"`
				From telegramUser `json:"from"`
			}{Text: text, From: tg}
			up.Message.Chat.ID = tgID
			up.Message.Chat.Type = "private"
		} else {
			up.Callback = &struct {
				ID      string       `json:"id"`
				From    telegramUser `json:"from"`
				Data    string       `json:"data"`
				Message struct {
					Chat struct {
						ID   int64  `json:"id"`
						Type string `json:"type"`
					} `json:"chat"`
				} `json:"message"`
			}{ID: "callback", From: tg, Data: data}
			up.Callback.Message.Chat.ID = tgID
			up.Callback.Message.Chat.Type = "private"
		}
		if e := a.handleUpdate(ctx, s, "12345:test-token", up); e != nil {
			t.Fatal(e)
		}
	}
	var link map[string]string
	json.Unmarshal(c.call("GET", "/api/auth/login-link", nil, 200), &link)
	c.call("POST", "/api/auth/exchange", map[string]string{"token": link["token"]}, 409)
	send("/start login_"+link["token"], "")
	send("", "role:intern")
	send("", "direction:backend")
	send("", "team:backend-platform")
	send("", "confirm_login")
	browser := &apiClient{t: t, h: a.routes()}
	var tgu User
	json.Unmarshal(browser.call("POST", "/api/auth/exchange", map[string]string{"token": link["token"]}, 200), &tgu)
	defer a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", tgu.ID)
	browser.call("POST", "/api/auth/exchange", map[string]string{"token": link["token"]}, 401)
	if tgu.IsDemo || tgu.TelegramID != tgID {
		t.Fatal("wrong Telegram profile", tgu)
	}
	browser.call("POST", "/api/auth/telegram", map[string]string{"init_data": signedTelegramData(tgID, "12345:test-token")}, 200)
	browser.call("POST", "/api/auth/telegram", map[string]string{"init_data": signedTelegramData(tgID+1, "12345:test-token")}, 409)
	browser.call("POST", "/api/auth/telegram", map[string]string{"init_data": signedTelegramData(tgID, "forged-token")}, 401)
	mentor, _ := demo(t, a, "mentor", tgu.TeamID)
	mentor.call("GET", "/api/interns/"+tgu.ID, nil, 403)
	browser.call("POST", "/api/meetings/demo", map[string]any{}, 200)
	a.sendReminders(ctx)
	meetings, _ := records[Meeting](ctx, a, "meeting", tgu.ID)
	sent := false
	for _, m := range meetings {
		if m.NotificationStatus == "sent" {
			sent = true
		}
	}
	if !sent {
		t.Fatal("reminder not sent")
	}
	pollCtx, cancel := context.WithCancel(ctx)
	polls := 0
	_, _ = a.db.Exec(ctx, "INSERT INTO runtime(key,value) VALUES($1,'37') ON CONFLICT(key) DO UPDATE SET value='37'", botOffsetKey("12345:test-token"))
	prior := a.http.Transport
	a.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.HasSuffix(r.URL.Path, "getUpdates") {
			polls++
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if polls == 1 {
				if body["offset"].(float64) != 37 {
					t.Error("did not load per-bot offset", body)
				}
				newToken, _ := a.cipher.seal("67890:another-test-token")
				a.db.Exec(ctx, "UPDATE settings SET telegram_secret=$1 WHERE id=1", newToken)
			} else {
				if body["offset"].(float64) != 0 {
					t.Error("new bot reused prior offset", body)
				}
				cancel()
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"ok":true,"result":[]}`)), Header: http.Header{}}, nil
		}
		return prior.RoundTrip(r)
	})}
	a.botLoop(pollCtx)
	if polls != 2 {
		t.Fatal("poller did not switch token", polls)
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	a.scheduler(stopped)
}
func TestPersistentSettingsSurviveBlankEnvironment(t *testing.T) {
	a := testApp(t)
	_ = a
	t.Setenv("AI_MOCK_MODE", "false")
	t.Setenv("ROUTERAI_API_KEY", "")
	t.Setenv("ROUTERAI_MODEL", "")
	other, e := newApp(context.Background(), os.Getenv("TEST_DATABASE_URL"), "test-password-very-long", "test-encryption-key-over-32-characters", "http://example.test", false)
	if e != nil {
		t.Fatal("stored settings must override empty bootstrap environment:", e)
	}
	other.db.Close()
}
func TestRAGAccessRetrieval(t *testing.T) {
	a := testApp(t)
	articles, e := a.searchArticles(context.Background(), "Как получить доступ к репозиторию?", "backend")
	if e != nil || len(articles) == 0 || articles[0].ID != "gitlab-access" {
		t.Fatalf("expected access instruction first; got %#v, %v", articles, e)
	}
}
func TestSeedIdempotence(t *testing.T) {
	a := testApp(t)
	ctx := context.Background()
	for _, tr := range tracks() {
		if _, e := a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", "seed-intern-"+tr.ID); e != nil {
			t.Fatal(e)
		}
	}
	a.db.Exec(ctx, "DELETE FROM users WHERE id IN ('seed-manager-backend-platform','seed-manager-frontend-web-platform')")
	if e := a.seedExpanded(ctx); e != nil {
		t.Fatal(e)
	}
	var before, after int
	a.db.QueryRow(ctx, "SELECT count(*) FROM records WHERE user_id LIKE 'seed-%'").Scan(&before)
	if e := a.seedExpanded(ctx); e != nil {
		t.Fatal(e)
	}
	a.db.QueryRow(ctx, "SELECT count(*) FROM records WHERE user_id LIKE 'seed-%'").Scan(&after)
	if before != after {
		t.Fatal("seed duplicated persisted records", before, after)
	}
	var articles, lessons int
	a.db.QueryRow(ctx, "SELECT count(*) FROM catalog WHERE kind='article'").Scan(&articles)
	a.db.QueryRow(ctx, "SELECT count(*) FROM catalog WHERE kind='lesson'").Scan(&lessons)
	if articles < 15 || lessons < 10 {
		t.Fatal("seed contents below spec", articles, lessons)
	}
}
func TestBotPollingRetriesTransientUpdates(t *testing.T) {
	a := testApp(t)
	ctx := context.Background()
	s, oldToken, oldKey, e := a.settings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	original := s
	s.TelegramEnabled = true
	s.TelegramUsername = "test_buddy_bot"
	s.AIMockMode = true
	s.RouterAIModel = ""
	token := "90001:poll-test-token"
	te, _ := a.cipher.seal(token)
	b, _ := json.Marshal(s)
	a.db.Exec(ctx, "UPDATE settings SET body=$1,telegram_secret=$2 WHERE id=1", b, te)
	defer func() {
		b, _ := json.Marshal(original)
		te, _ := a.cipher.seal(oldToken)
		ke, _ := a.cipher.seal(oldKey)
		a.db.Exec(ctx, "UPDATE settings SET body=$1,telegram_secret=$2,routerai_secret=$3 WHERE id=1", b, te, ke)
	}()
	for _, permanent := range []bool{false, true} {
		pollCtx, cancel := context.WithCancel(ctx)
		polls, sends := 0, 0
		a.db.Exec(ctx, "DELETE FROM runtime WHERE key=$1", botOffsetKey(token))
		a.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			status, body := 200, `{"ok":true,"result":{}}`
			if strings.HasSuffix(r.URL.Path, "getUpdates") {
				polls++
				if polls == 1 {
					body = `{"ok":true,"result":[{"update_id":77,"message":{"text":"/help","chat":{"id":923456,"type":"private"},"from":{"id":923456,"first_name":"Test"}}}]}`
				} else {
					cancel()
					body = `{"ok":true,"result":[]}`
				}
			} else if strings.HasSuffix(r.URL.Path, "sendMessage") {
				sends++
				if permanent || sends < 3 {
					status = 503
				}
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
		})}
		a.botLoop(pollCtx)
		if sends != 3 {
			t.Fatal("update wasn't retried three times", permanent, sends)
		}
		var offset string
		a.db.QueryRow(ctx, "SELECT value FROM runtime WHERE key=$1", botOffsetKey(token)).Scan(&offset)
		if offset != "78" {
			t.Fatal("offset advanced incorrectly", offset)
		}
	}
}
func TestRoleAndInputBoundaries(t *testing.T) {
	a := testApp(t)
	ctx := context.Background()
	c, u := demo(t, a, "intern", "")
	defer a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID)
	c.call("GET", "/api/interns", nil, 403)
	c.call("POST", "/api/requests/no-such/decision", map[string]bool{"approve": true}, 403)
	c.call("POST", "/api/auth/demo", map[string]string{"role": "admin"}, 400)
	c.call("POST", "/api/auth/demo", map[string]string{"role": "intern", "team_id": "no-team"}, 400)
	c.call("POST", "/api/auth/demo", map[string]string{"role": "intern", "name": strings.Repeat("a", 101)}, 400)
	c.call("POST", "/api/auth/telegram", map[string]string{"init_data": "forged"}, 503)
	c.call("GET", "/api/auth/login-link", nil, 503)
	mentor, _ := demo(t, a, "mentor", u.TeamID)
	mentor.call("POST", "/api/requests", map[string]any{"resource": "GitLab", "reason": "test", "confirmed": true}, 403)
	mentor.call("POST", "/api/lessons/lesson-backend/exam", map[string]any{"answers": map[string]int{}}, 403)
	mentor.call("POST", "/api/interns/"+u.ID+"/tasks", map[string]any{"title": ""}, 400)
	mentor.call("POST", "/api/interns/"+u.ID+"/tasks", map[string]any{"title": "test", "due_at": time.Now().Add(time.Hour), "article_id": "missing"}, 400)
	mentor.call("POST", "/api/interns/"+u.ID+"/meetings", map[string]any{"title": "test", "starts_at": time.Now().Add(time.Hour), "article_id": "missing"}, 400)
	mentor.call("POST", "/api/interns/"+u.ID+"/meetings", map[string]any{}, 400)
	foreign, fu := demo(t, a, "manager", "frontend-web-platform")
	defer a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", fu.ID)
	foreign.call("GET", "/api/interns/"+u.ID, nil, 403)
	t.Setenv("DEMO_MODE", "false")
	c.call("POST", "/api/auth/demo", map[string]string{"role": "intern"}, 403)
	t.Setenv("DEMO_MODE", "true")
	a.secure = true
	secureClient := &apiClient{t: t, h: a.routes()}
	secureClient.call("POST", "/api/admin/login", map[string]string{"password": "test-password-very-long"}, 200)
	if !secureClient.cookie.Secure || !secureClient.cookie.HttpOnly || secureClient.cookie.SameSite != http.SameSiteNoneMode {
		t.Fatal("unsafe embedded-app cookie", secureClient.cookie)
	}
	a.secure = false
	// Real HTTP decoder/content checks, expired cookie and rate-limit boundary.
	r := httptest.NewRequest("POST", "/api/auth/demo", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	a.routes().ServeHTTP(w, r)
	if w.Code != 415 {
		t.Fatal("accepted wrong content type", w.Code)
	}
	r = httptest.NewRequest("POST", "/api/auth/demo", strings.NewReader("{"))
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	a.routes().ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatal("accepted malformed JSON", w.Code)
	}
	expired := &apiClient{t: t, h: a.routes(), cookie: &http.Cookie{Name: "buddy_session", Value: "invalid"}}
	expired.call("GET", "/api/me", nil, 401)
	admin := &apiClient{t: t, h: a.routes()}
	for i := 0; i < 10; i++ {
		admin.call("POST", "/api/admin/login", map[string]string{"password": "wrong"}, 401)
	}
	admin.call("POST", "/api/admin/login", map[string]string{"password": "test-password-very-long"}, 429)
}
func TestProviderFailureBoundaries(t *testing.T) {
	a := testApp(t)
	ctx := context.Background()
	c, u := demo(t, a, "intern", "")
	defer a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID)
	s, oldToken, oldKey, e := a.settings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	original := s
	s.AIMockMode = false
	s.RouterAIModel = "failure-test-model"
	te, _ := a.cipher.seal("12345:test-token")
	ke, _ := a.cipher.seal("failure-test-key")
	b, _ := json.Marshal(s)
	a.db.Exec(ctx, "UPDATE settings SET body=$1,telegram_secret=$2,routerai_secret=$3 WHERE id=1", b, te, ke)
	defer func() {
		b, _ := json.Marshal(original)
		te, _ := a.cipher.seal(oldToken)
		ke, _ := a.cipher.seal(oldKey)
		a.db.Exec(ctx, "UPDATE settings SET body=$1,telegram_secret=$2,routerai_secret=$3 WHERE id=1", b, te, ke)
	}()
	for _, body := range []string{`not JSON`, `{"choices":[]}`, `{"choices":[{"message":{"role":"assistant","content":""}}]}`, `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"x","type":"function","function":{"name":"unknown","arguments":"not-json"}}]}}]}`} {
		a.http = fakeHTTP(t, func(r *http.Request) string { return body })
		c.call("POST", "/api/chat", map[string]string{"message": "Помоги с Git"}, 502)
	}
	a.http = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 429, Body: io.NopCloser(strings.NewReader("upstream error containing a secret")), Header: http.Header{}}, nil
	})}
	body := c.call("POST", "/api/chat", map[string]string{"message": "Объясни API"}, 502)
	if strings.Contains(string(body), "upstream error containing") {
		t.Fatal("provider raw error leaked")
	}
	admin := &apiClient{t: t, h: a.routes()}
	admin.call("POST", "/api/admin/login", map[string]string{"password": "test-password-very-long"}, 200)
	admin.call("POST", "/api/admin/test/routerai", map[string]any{}, 200)
	admin.call("POST", "/api/admin/test/telegram", map[string]any{}, 200)
	a.http = fakeHTTP(t, func(r *http.Request) string {
		if strings.HasSuffix(r.URL.Path, "/models") {
			return `{"data":[{"id":"failure-test-model"}]}`
		}
		return `{"choices":[{"message":{"role":"assistant","content":"I cannot call tools"}}]}`
	})
	body = admin.call("POST", "/api/admin/test/routerai", map[string]any{}, 200)
	var result map[string]any
	json.Unmarshal(body, &result)
	if result["ok"] != false {
		t.Fatal("unsupported tool model accepted")
	}
	var res aiResponse
	if e := a.routerCall(ctx, Settings{RouterAIBaseURL: "https://attacker.test"}, "key", "GET", "/models", nil, &res); e == nil {
		t.Fatal("unexpected endpoint allowed")
	}
}

func TestStartupSecurityConfiguration(t *testing.T) {
	ctx := context.Background()
	if _, e := newApp(ctx, "", "short", "a-long-encryption-key-over-thirty-two-chars", "", false); e == nil {
		t.Fatal("missing admin password accepted")
	}
	if _, e := newApp(ctx, "", "long-enough-password", "short", "", false); e == nil {
		t.Fatal("missing encryption key accepted")
	}
}

func TestDeleteTelegramProfileRemovesRegistrationState(t *testing.T) {
	a := testApp(t)
	ctx := context.Background()
	tgID := time.Now().UnixNano()%1000000000 + 10000
	u, e := a.createUser(ctx, "Удаляемый Telegram профиль", "intern", "backend-platform", tgID, false)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		a.db.Exec(ctx, "DELETE FROM users WHERE id=$1", u.ID)
		a.db.Exec(ctx, "DELETE FROM bot_states WHERE telegram_id IN ($1,$2)", tgID, tgID+1)
	})
	pendingToken, confirmedToken := id(), id()
	t.Cleanup(func() {
		a.db.Exec(ctx, "DELETE FROM login_links WHERE token_hash IN ($1,$2)", hashToken(pendingToken), hashToken(confirmedToken))
	})
	for _, link := range []struct {
		token string
		owner any
	}{{pendingToken, nil}, {confirmedToken, u.ID}} {
		if _, e = a.db.Exec(ctx, "INSERT INTO login_links(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '10 minutes')", hashToken(link.token), link.owner); e != nil {
			t.Fatal(e)
		}
	}
	if e = a.saveBotState(ctx, tgID, botState{Name: u.Name, Role: "intern", LoginToken: pendingToken}); e != nil {
		t.Fatal(e)
	}
	if e = a.saveBotState(ctx, tgID+1, botState{Name: "Другой пользователь", Role: "mentor"}); e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/auth/telegram", nil)
	if e = a.newSession(w, r, u.ID, false); e != nil {
		t.Fatal(e)
	}
	cookie := w.Result().Cookies()[0]
	originalSessionHash := hashToken(cookie.Value)
	c := &apiClient{t: t, h: a.routes(), cookie: cookie}
	c.call("DELETE", "/api/me", nil, 200)
	c.call("GET", "/api/me", nil, 401)
	for _, check := range []struct {
		name, sql string
		args      []any
	}{
		{"user", "SELECT count(*) FROM users WHERE id=$1", []any{u.ID}},
		{"registration state", "SELECT count(*) FROM bot_states WHERE telegram_id=$1", []any{tgID}},
		{"session", "SELECT count(*) FROM sessions WHERE token_hash=$1", []any{originalSessionHash}},
		{"tasks and scheduled reminders", "SELECT count(*) FROM records WHERE user_id=$1", []any{u.ID}},
		{"pending and confirmed login links", "SELECT count(*) FROM login_links WHERE token_hash IN ($1,$2)", []any{hashToken(pendingToken), hashToken(confirmedToken)}},
	} {
		var n int
		if e = a.db.QueryRow(ctx, check.sql, check.args...).Scan(&n); e != nil || n != 0 {
			t.Fatalf("retained %s: count=%d error=%v", check.name, n, e)
		}
	}
	var n int
	if e = a.db.QueryRow(ctx, "SELECT count(*) FROM bot_states WHERE telegram_id=$1 AND body->>'name'='Другой пользователь'", tgID+1).Scan(&n); e != nil || n != 1 {
		t.Fatal("unrelated bot registration state changed", n, e)
	}
	if _, e = a.createUser(ctx, "Повторная регистрация", "intern", "backend-platform", tgID, false); e != nil {
		t.Fatal("Telegram ID cannot register again", e)
	}
	a.db.Exec(ctx, "DELETE FROM users WHERE telegram_id=$1", tgID)
}
