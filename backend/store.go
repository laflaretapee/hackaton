package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.sql
var migration string

type App struct {
	db                    *pgxpool.Pool
	cipher                *secretCipher
	adminPassword, appURL string
	secure                bool
	http                  *http.Client
	mu                    sync.Mutex
	attempts              map[string]loginAttempt
	chatWindows           map[string]loginAttempt
	chatActive            int
}
type loginAttempt struct {
	count int
	since time.Time
}

func newApp(ctx context.Context, database, password, key, appURL string, secure bool) (*App, error) {
	if len(password) < 12 {
		return nil, errors.New("ADMIN_PASSWORD must contain at least 12 characters")
	}
	c, e := newCipher(key)
	if e != nil {
		return nil, e
	}
	db, e := pgxpool.New(ctx, database)
	if e != nil {
		return nil, e
	}
	a := &App{db: db, cipher: c, adminPassword: password, appURL: strings.TrimSuffix(appURL, "/"), secure: secure, http: &http.Client{Timeout: 125 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}, attempts: map[string]loginAttempt{}}
	if _, e = db.Exec(ctx, migration); e != nil {
		db.Close()
		return nil, e
	}
	if e = a.seed(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return a, nil
}
func (a *App) loadUser(ctx context.Context, userID string) (User, error) {
	var u User
	e := a.db.QueryRow(ctx, `SELECT u.id,u.name,u.role,u.team_id,t.name,t.direction,coalesce(u.mentor_id,''),coalesce(m.name,''),u.notifications,u.reminder_minutes,coalesce(u.telegram_id,0),u.is_demo FROM users u JOIN teams t ON t.id=u.team_id LEFT JOIN users m ON m.id=u.mentor_id WHERE u.id=$1`, userID).Scan(&u.ID, &u.Name, &u.Role, &u.TeamID, &u.TeamName, &u.Direction, &u.MentorID, &u.MentorName, &u.Notifications, &u.ReminderMinutes, &u.TelegramID, &u.IsDemo)
	return u, e
}
func (a *App) teams(ctx context.Context) ([]Team, error) {
	rows, e := a.db.Query(ctx, "SELECT id,name,direction FROM teams ORDER BY direction,name")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Team{}
	for rows.Next() {
		var t Team
		if e = rows.Scan(&t.ID, &t.Name, &t.Direction); e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func (a *App) save(ctx context.Context, kind, owner, key string, value any) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	_, e = a.db.Exec(ctx, `INSERT INTO records(id,kind,user_id,body) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET body=excluded.body WHERE records.user_id=excluded.user_id AND records.kind=excluded.kind`, key, kind, owner, b)
	return e
}
func records[T any](ctx context.Context, a *App, kind, owner string) ([]T, error) {
	rows, e := a.db.Query(ctx, "SELECT body FROM records WHERE kind=$1 AND user_id=$2 ORDER BY created_at,id", kind, owner)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var b []byte
		var x T
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func record[T any](ctx context.Context, a *App, kind, key string) (T, string, error) {
	var b []byte
	var owner string
	var x T
	e := a.db.QueryRow(ctx, "SELECT body,user_id FROM records WHERE id=$1 AND kind=$2", key, kind).Scan(&b, &owner)
	if e != nil {
		return x, "", e
	}
	e = json.Unmarshal(b, &x)
	return x, owner, e
}
func catalog[T any](ctx context.Context, a *App, kind string) ([]T, error) {
	rows, e := a.db.Query(ctx, "SELECT body FROM catalog WHERE kind=$1 ORDER BY id", kind)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var b []byte
		var x T
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		if e = json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (a *App) article(ctx context.Context, key string) (Article, error) {
	var b []byte
	var x Article
	e := a.db.QueryRow(ctx, "SELECT body FROM catalog WHERE id=$1 AND kind='article'", key).Scan(&b)
	if e == nil {
		e = json.Unmarshal(b, &x)
	}
	return x, e
}
func (a *App) lesson(ctx context.Context, key string) (Lesson, error) {
	var b []byte
	var x Lesson
	e := a.db.QueryRow(ctx, "SELECT body FROM catalog WHERE id=$1 AND kind='lesson'", key).Scan(&b)
	if e == nil {
		e = json.Unmarshal(b, &x)
	}
	return x, e
}
func (a *App) settings(ctx context.Context) (Settings, string, string, error) {
	var b []byte
	var t, k string
	var s Settings
	e := a.db.QueryRow(ctx, "SELECT body,telegram_secret,routerai_secret FROM settings WHERE id=1").Scan(&b, &t, &k)
	if e != nil {
		return s, "", "", e
	}
	if e = json.Unmarshal(b, &s); e != nil {
		return s, "", "", e
	}
	token, e := a.cipher.open(t)
	if e != nil {
		return s, "", "", e
	}
	key, e := a.cipher.open(k)
	s.TelegramTokenSet = token != ""
	s.RouterAIKeySet = key != ""
	return s, token, key, e
}
func (a *App) audit(ctx context.Context, action string) {
	_, _ = a.db.Exec(ctx, "INSERT INTO audit(action) VALUES($1)", action)
}
func (a *App) createUser(ctx context.Context, name, role, teamID string, telegramID int64, demo bool) (User, error) {
	return a.createUserID(ctx, name, role, teamID, telegramID, demo, "")
}
func (a *App) createUserID(ctx context.Context, name, role, teamID string, telegramID int64, demo bool, uid string) (User, error) {
	var empty User
	if role != "intern" && role != "mentor" && role != "manager" {
		return empty, errors.New("invalid role")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = map[string]string{"intern": "Александр", "mentor": "Мария · наставник", "manager": "Андрей · руководитель"}[role]
	}
	if len([]rune(name)) > 100 {
		return empty, errors.New("name too long")
	}
	if teamID == "" {
		teamID = "backend-platform"
	}
	var exists bool
	if e := a.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE id=$1)", teamID).Scan(&exists); e != nil || !exists {
		return empty, errors.New("team does not exist")
	}
	if demo && role == "mentor" {
		return a.loadUser(ctx, "demo-mentor-"+teamID)
	}
	if uid == "" {
		uid = id()
	}
	mentor := ""
	if role == "intern" {
		_ = a.db.QueryRow(ctx, "SELECT id FROM users WHERE role='mentor' AND team_id=$1 AND is_demo=$2 ORDER BY created_at LIMIT 1", teamID, demo).Scan(&mentor)
	}
	var tg any
	if telegramID > 0 {
		tg = telegramID
	}
	var m any
	if mentor != "" {
		m = mentor
	}
	tx, e := a.db.Begin(ctx)
	if e != nil {
		return empty, e
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, "INSERT INTO users(id,name,role,team_id,mentor_id,telegram_id,is_demo) VALUES($1,$2,$3,$4,$5,$6,$7)", uid, name, role, teamID, m, tg, demo)
	if e != nil {
		return empty, e
	}
	if role == "intern" {
		var dir string
		_ = tx.QueryRow(ctx, "SELECT direction FROM teams WHERE id=$1", teamID).Scan(&dir)
		tasks := []Task{{ID: id(), Title: "Познакомиться с командой", Description: "Представьтесь в учебном чате, уточните время синков и договоритесь о первой встрече с наставником.", Status: "not_started", Kind: "manual", DueAt: time.Now().Add(24 * time.Hour), ArticleID: "team-process", Priority: "high"}, {ID: id(), Title: "Изучить Git и Code Review", Description: "Прочитайте инструкцию и подготовьте небольшой pull request в учебном репозитории.", Status: "not_started", Kind: "manual", DueAt: time.Now().Add(48 * time.Hour), ArticleID: "git-review", Priority: "medium"}, {ID: id(), Title: "Получить доступ к GitLab", Description: "Создайте демонстрационную заявку и дождитесь согласования наставника. Реальные права не выдаются.", Status: "not_started", Kind: "access", DueAt: time.Now().Add(48 * time.Hour), ArticleID: "gitlab-access", Priority: "high"}, {ID: id(), Title: "Пройти проверку по направлению", Description: "Пройдите экзамен по вашему направлению с результатом не ниже 80%.", Status: "not_started", Kind: "exam", DueAt: time.Now().Add(72 * time.Hour), ArticleID: "track-" + dir, Priority: "medium"}}
		for _, t := range tasks {
			b, _ := json.Marshal(t)
			if _, e = tx.Exec(ctx, "INSERT INTO records(id,kind,user_id,body) VALUES($1,'task',$2,$3)", t.ID, uid, b); e != nil {
				return empty, e
			}
		}
		meeting := Meeting{ID: id(), Title: "Знакомство с наставником и Git", StartsAt: nextWorkingMeeting(time.Now()), ArticleID: "git-review", NotificationStatus: "pending"}
		b, _ := json.Marshal(meeting)
		if _, e = tx.Exec(ctx, "INSERT INTO records(id,kind,user_id,body) VALUES($1,'meeting',$2,$3)", meeting.ID, uid, b); e != nil {
			return empty, e
		}
	}
	if e = tx.Commit(ctx); e != nil {
		return empty, e
	}
	return a.loadUser(ctx, uid)
}
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func envint(key string, fallback int) int {
	n, e := strconv.Atoi(os.Getenv(key))
	if e != nil {
		return fallback
	}
	return n
}
func nextWorkingMeeting(now time.Time) time.Time {
	local := now.In(time.Local).AddDate(0, 0, 1)
	d := time.Date(local.Year(), local.Month(), local.Day(), 11, 0, 0, 0, time.Local)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, 1)
	}
	return d.UTC()
}
