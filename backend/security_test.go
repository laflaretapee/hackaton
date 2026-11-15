package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestEncryptRoundTripAndTampering(t *testing.T) {
	c, err := newCipher("a-secure-server-key-with-32-characters")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := c.seal("top-secret")
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "top-secret" {
		t.Fatal("stored plaintext")
	}
	got, err := c.open(encrypted)
	if err != nil || got != "top-secret" {
		t.Fatal(got, err)
	}
	if _, err = c.open(encrypted[:len(encrypted)-4] + "xxxx"); err == nil {
		t.Fatal("accepted tampered ciphertext")
	}
	if _, err = newCipher("short"); err == nil {
		t.Fatal("accepted weak key")
	}
}
func TestTelegramSignatureFreshnessAndTampering(t *testing.T) {
	token := "test-token"
	v := url.Values{"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)}, "user": {`{"id":123,"first_name":"Ира"}`}, "signature": {"telegram-signature"}}
	check := "auth_date=" + v.Get("auth_date") + "\nsignature=" + v.Get("signature") + "\nuser=" + v.Get("user")
	k := hmac.New(sha256.New, []byte("WebAppData"))
	k.Write([]byte(token))
	h := hmac.New(sha256.New, k.Sum(nil))
	h.Write([]byte(check))
	v.Set("hash", hex.EncodeToString(h.Sum(nil)))
	user, err := verifyTelegram(v.Encode(), token, time.Now())
	if err != nil || user.ID != 123 {
		t.Fatal(user, err)
	}
	v.Set("user", `{"id":999}`)
	if _, err = verifyTelegram(v.Encode(), token, time.Now()); err == nil {
		t.Fatal("accepted forged identity")
	}
	v.Set("auth_date", "1")
	if _, err = verifyTelegram(v.Encode(), token, time.Now()); err == nil {
		t.Fatal("accepted expired identity")
	}
}
func TestScopeIsolation(t *testing.T) {
	mentor := User{ID: "m", Role: "mentor", TeamID: "t"}
	intern := User{ID: "i", Role: "intern", TeamID: "t", MentorID: "m"}
	if !canView(mentor, intern) {
		t.Fatal("assigned mentor denied")
	}
	intern.MentorID = "other"
	if canView(mentor, intern) {
		t.Fatal("unassigned mentor allowed")
	}
	manager := User{ID: "boss", Role: "manager", TeamID: "t"}
	if !canView(manager, intern) {
		t.Fatal("team manager denied")
	}
	intern.TeamID = "foreign"
	if canView(manager, intern) {
		t.Fatal("foreign team exposed")
	}
	if canView(User{ID: "i2", Role: "intern"}, intern) {
		t.Fatal("intern exposed another profile")
	}
}
func TestSettingsValidationRejectsSSRFAndMissingSecrets(t *testing.T) {
	s := Settings{RouterAIBaseURL: "http://127.0.0.1:1234", AIMockMode: true, AITimeoutSeconds: 60, AIMaxTokens: 1500}
	if validateSettings(s, "", "") == nil {
		t.Fatal("SSRF allowed")
	}
	s.RouterAIBaseURL = "https://routerai.ru/api/v1"
	s.TelegramEnabled = true
	if validateSettings(s, "", "") == nil {
		t.Fatal("enabled without token")
	}
	s.TelegramEnabled = false
	s.AIMockMode = false
	if validateSettings(s, "", "") == nil {
		t.Fatal("live AI without key")
	}
	s.AIMockMode = true
	if err := validateSettings(s, "", ""); err != nil {
		t.Fatal(err)
	}
}
func TestExamGradesServerAnswers(t *testing.T) {
	qs := []Question{{ID: "q1", Prompt: "Why review?", Options: []string{"quality", "speed"}, Correct: 0, Explanation: "catch defects"}, {ID: "q2", Prompt: "Which branch?", Options: []string{"main", "feature"}, Correct: 1, Explanation: "isolation"}}
	a, err := gradeExam("lesson", qs, map[string]int{"q1": 0, "q2": 0})
	if err != nil || a.Score != 50 || a.Correct != 1 || a.Total != 2 {
		t.Fatal(a, err)
	}
	if _, err = gradeExam("lesson", qs, map[string]int{"q1": 0}); err == nil {
		t.Fatal("missing answer accepted")
	}
	if _, err = gradeExam("lesson", qs, map[string]int{"q1": 4, "q2": 0}); err == nil {
		t.Fatal("invalid option accepted")
	}
}
func TestChatRateLimits(t *testing.T) {
	a := &App{}
	now := time.Now()
	for i := 0; i < 12; i++ {
		if !a.allowChat("user", now) {
			t.Fatal("limited too early", i)
		}
	}
	if a.allowChat("user", now) {
		t.Fatal("unlimited per-user requests")
	}
	if !a.allowChat("user", now.Add(2*time.Minute)) {
		t.Fatal("window did not reset")
	}
}
func signedTelegramData(tgID int64, token string) string {
	v := url.Values{"auth_date": {strconv.FormatInt(time.Now().Unix(), 10)}, "user": {`{"id":` + strconv.FormatInt(tgID, 10) + `,"first_name":"Тест"}`}}
	check := "auth_date=" + v.Get("auth_date") + "\nuser=" + v.Get("user")
	k := hmac.New(sha256.New, []byte("WebAppData"))
	k.Write([]byte(token))
	h := hmac.New(sha256.New, k.Sum(nil))
	h.Write([]byte(check))
	v.Set("hash", hex.EncodeToString(h.Sum(nil)))
	return v.Encode()
}
func TestWeekendMeetingScheduling(t *testing.T) {
	friday := time.Date(2026, 11, 13, 19, 0, 0, 0, time.Local)
	meeting := nextWorkingMeeting(friday).In(time.Local)
	if meeting.Weekday() != time.Monday || meeting.Hour() != 11 {
		t.Fatal(meeting)
	}
}
