package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHTTPAuthBoundary(t *testing.T) {
	a := &App{appURL: "https://buddy.test"}
	h := a.routes()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/me", nil))
	if w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/auth/demo", bytes.NewBufferString(`{"role":"intern"}`))
	r.Header.Set("Origin", "https://attacker.test")
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal(w.Code)
	}
}
func TestIntegrationJourney(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL journey")
	}
	a, e := newApp(context.Background(), url, "test-password-very-long", "test-encryption-key-over-32-characters", "http://example.test", false)
	if e != nil {
		t.Fatal(e)
	}
	defer a.db.Close()
	h := a.routes()
	call := func(method, path string, data any, cookie *http.Cookie) (int, []byte, *http.Cookie) {
		var b bytes.Buffer
		if data != nil {
			json.NewEncoder(&b).Encode(data)
		}
		r := httptest.NewRequest(method, path, &b)
		r.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		var c *http.Cookie
		if len(w.Result().Cookies()) > 0 {
			c = w.Result().Cookies()[0]
		}
		return w.Code, w.Body.Bytes(), c
	}
	status, body, c := call("POST", "/api/auth/demo", map[string]any{"role": "intern", "name": "Integration intern"}, nil)
	if status != 200 || c == nil {
		t.Fatal(status, string(body))
	}
	var user User
	json.Unmarshal(body, &user)
	status, body, _ = call("GET", "/api/tasks", nil, c)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	var tasks []Task
	json.Unmarshal(body, &tasks)
	if len(tasks) < 4 {
		t.Fatal(string(body))
	}
	for _, task := range tasks {
		if task.Kind == "access" {
			s, _, _ := call("PATCH", "/api/tasks/"+task.ID, map[string]string{"status": "completed"}, c)
			if s != 409 {
				t.Fatal("access manually complete", s)
			}
		}
	}
	status, body, _ = call("POST", "/api/requests", map[string]any{"resource": "GitLab", "reason": "Первая учебная задача", "confirmed": true}, c)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	var req Request
	json.Unmarshal(body, &req)
	_, _, mc := call("POST", "/api/auth/demo", map[string]string{"role": "mentor", "team_id": user.TeamID}, nil)
	status, body, _ = call("POST", "/api/requests/"+req.ID+"/decision", map[string]bool{"approve": true}, mc)
	if status != 200 {
		t.Fatal(status, string(body))
	}

	status, body, _ = call("POST", "/api/requests", map[string]any{"resource": "GitLab", "reason": "Повторная проверка", "confirmed": true}, c)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	var rejected Request
	json.Unmarshal(body, &rejected)
	status, body, _ = call("POST", "/api/requests/"+rejected.ID+"/decision", map[string]bool{"approve": false}, mc)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	_, body, _ = call("GET", "/api/tasks", nil, c)
	json.Unmarshal(body, &tasks)
	for _, task := range tasks {
		if task.Kind == "access" && task.Status != "completed" {
			t.Fatal("a later rejection regressed granted access", task.Status)
		}
	}
	status, body, _ = call("GET", "/api/lessons/lesson-backend/exam", nil, c)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	var exam map[string]any
	json.Unmarshal(body, &exam)
	if len(exam["questions"].([]any)) != 5 {
		t.Fatal("exam lost persisted questions", string(body))
	}
	for _, q := range exam["questions"].([]any) {
		if _, ok := q.(map[string]any)["correct"]; ok {
			t.Fatal("answers leaked")
		}
	}
	status, body, _ = call("POST", "/api/lessons/lesson-backend/exam", map[string]any{"answers": map[string]int{"q1": 1, "q2": 0, "q3": 1, "q4": 2, "q5": 0}}, c)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	var at Attempt
	json.Unmarshal(body, &at)
	if at.Score != 100 {
		t.Fatal("persisted answer grading", string(body))
	}
	status, body, _ = call("GET", "/api/interns/"+user.ID, nil, mc)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	_, _, other := call("POST", "/api/auth/demo", map[string]string{"role": "intern"}, nil)
	status, _, _ = call("GET", "/api/interns/"+user.ID, nil, other)
	if status != 403 {
		t.Fatal("cross-user access", status)
	}
	status, body, _ = call("POST", "/api/admin/login", map[string]string{"password": "test-password-very-long"}, nil)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	status, body, _ = call("DELETE", "/api/me", nil, c)
	if status != 200 {
		t.Fatal(status, string(body))
	}
	status, _, _ = call("GET", "/api/me", nil, c)
	if status != 401 {
		t.Fatal("session survived deletion", status)
	}
}
