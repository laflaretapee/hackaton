package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type secretCipher struct{ aead cipher.AEAD }

func newCipher(key string) (*secretCipher, error) {
	if len(key) < 32 {
		return nil, errors.New("ENCRYPTION_KEY must contain at least 32 characters")
	}
	sum := sha256.Sum256([]byte(key))
	b, e := aes.NewCipher(sum[:])
	if e != nil {
		return nil, e
	}
	a, e := cipher.NewGCM(b)
	return &secretCipher{a}, e
}
func (c *secretCipher) seal(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	n := make([]byte, c.aead.NonceSize())
	if _, e := rand.Read(n); e != nil {
		return "", e
	}
	return base64.RawStdEncoding.EncodeToString(c.aead.Seal(n, n, []byte(plain), nil)), nil
}
func (c *secretCipher) open(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, e := base64.RawStdEncoding.DecodeString(encoded)
	if e != nil || len(raw) < c.aead.NonceSize() {
		return "", errors.New("invalid encrypted secret")
	}
	p, e := c.aead.Open(nil, raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():], nil)
	return string(p), e
}
func id() string {
	b := make([]byte, 24)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func hashToken(t string) string { s := sha256.Sum256([]byte(t)); return hex.EncodeToString(s[:]) }
func equalSecret(a, b string) bool {
	aSum, bSum := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aSum[:], bSum[:]) == 1
}

type telegramUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

func verifyTelegram(raw, token string, now time.Time) (telegramUser, error) {
	var user telegramUser
	if token == "" {
		return user, errors.New("Telegram integration is not configured")
	}
	v, e := url.ParseQuery(raw)
	if e != nil {
		return user, e
	}
	for _, x := range v {
		if len(x) != 1 {
			return user, errors.New("duplicate Telegram fields")
		}
	}
	given, e := hex.DecodeString(v.Get("hash"))
	if e != nil || len(given) != 32 {
		return user, errors.New("invalid Telegram signature")
	}
	v.Del("hash")
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := []string{}
	for _, k := range keys {
		lines = append(lines, k+"="+v.Get(k))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(token))
	h := hmac.New(sha256.New, secret.Sum(nil))
	h.Write([]byte(strings.Join(lines, "\n")))
	if !hmac.Equal(h.Sum(nil), given) {
		return user, errors.New("invalid Telegram signature")
	}
	ts, e := strconv.ParseInt(v.Get("auth_date"), 10, 64)
	if e != nil || now.Sub(time.Unix(ts, 0)) > 15*time.Minute || time.Unix(ts, 0).After(now.Add(time.Minute)) {
		return user, errors.New("Telegram authorization expired")
	}
	e = json.Unmarshal([]byte(v.Get("user")), &user)
	if e != nil || user.ID <= 0 {
		return user, errors.New("invalid Telegram user")
	}
	return user, nil
}
func canView(actor, target User) bool {
	if actor.IsDemo != target.IsDemo {
		return false
	}
	if actor.ID == target.ID {
		return true
	}
	if target.Role != "intern" {
		return false
	}
	return actor.Role == "mentor" && target.MentorID == actor.ID || actor.Role == "manager" && target.TeamID == actor.TeamID
}
func validateSettings(s Settings, telegram, key string) error {
	u, e := url.Parse(s.RouterAIBaseURL)
	if e != nil || u.Scheme != "https" || u.Host != "routerai.ru" || strings.TrimSuffix(u.Path, "/") != "/api/v1" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return errors.New("RouterAI URL must be https://routerai.ru/api/v1")
	}
	if s.TelegramEnabled && (telegram == "" || s.TelegramUsername == "") {
		return errors.New("Telegram token and username are required")
	}
	if !s.AIMockMode && (key == "" || strings.TrimSpace(s.RouterAIModel) == "") {
		return errors.New("RouterAI key and model are required for live mode")
	}
	if s.AITimeoutSeconds < 5 || s.AITimeoutSeconds > 120 || s.AIMaxTokens < 100 || s.AIMaxTokens > 8000 {
		return errors.New("AI timeout must be 5–120 seconds and max tokens 100–8000")
	}
	if s.AppURL != "" {
		a, e := url.Parse(s.AppURL)
		if e != nil || a.Host == "" || (a.Scheme != "https" && a.Scheme != "http") || a.User != nil {
			return errors.New("invalid app URL")
		}
	}
	return nil
}
func gradeExam(lesson string, questions []Question, answers map[string]int) (Attempt, error) {
	a := Attempt{ID: id(), LessonID: lesson, Total: len(questions), CreatedAt: time.Now().UTC(), Feedback: []Feedback{}}
	if len(answers) != len(questions) {
		return a, errors.New("answer all questions")
	}
	for _, q := range questions {
		i, ok := answers[q.ID]
		if !ok || i < 0 || i >= len(q.Options) {
			return a, fmt.Errorf("invalid answer for %s", q.ID)
		}
		correct := i == q.Correct
		if correct {
			a.Correct++
		}
		a.Feedback = append(a.Feedback, Feedback{q.Prompt, correct, q.Options[q.Correct], q.Explanation})
	}
	if a.Total > 0 {
		a.Score = a.Correct * 100 / a.Total
	}
	a.Recommendation = "Повторите вопросы с ошибками и перечитайте связанные разделы перед новой попыткой."
	if a.Score >= 80 {
		a.Recommendation = "Отличный результат. Примените знания в первой задаче и обсудите решение с наставником."
	}
	return a, nil
}
