package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (a *App) listTasks(w http.ResponseWriter, r *http.Request) error {
	x, e := records[Task](r.Context(), a, "task", current(r).ID)
	if e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) patchTask(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Status string `json:"status"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	t, owner, e := record[Task](r.Context(), a, "task", r.PathValue("id"))
	if e != nil {
		return e
	}
	if owner != current(r).ID {
		return problem(403, "Это чужая задача")
	}
	if t.Kind != "manual" {
		return problem(409, "Эта задача завершается после подтверждения сервиса или экзамена")
	}
	if in.Status != "in_progress" && in.Status != "completed" {
		return problem(400, "Недопустимый статус")
	}
	t.Status = in.Status
	if e = a.save(r.Context(), "task", owner, t.ID, t); e != nil {
		return e
	}
	return respond(w, t)
}
func (a *App) knowledge(w http.ResponseWriter, r *http.Request) error {
	out, e := a.searchArticles(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("direction"))
	if e != nil {
		return e
	}
	reads, e := records[map[string]string](r.Context(), a, "read", current(r).ID)
	if e != nil {
		return e
	}
	for i := range out {
		for _, v := range reads {
			if v["article_id"] == out[i].ID {
				out[i].Read = true
			}
		}
	}
	return respond(w, out)
}
func (a *App) searchArticles(ctx context.Context, query, direction string) ([]Article, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) > 200 {
		return nil, problem(400, "Слишком длинный поисковый запрос")
	}
	out, e := a.queryArticles(ctx, query, direction)
	if e != nil || len(out) > 0 || query == "" {
		return out, e
	}
	terms := strings.Fields(query)
	if len(terms) > 1 {
		return a.queryArticles(ctx, strings.Join(terms, " OR "), direction)
	}
	return out, nil
}
func (a *App) queryArticles(ctx context.Context, query, direction string) ([]Article, error) {
	rows, e := a.db.Query(ctx, `SELECT body FROM catalog WHERE kind='article' AND ($2='' OR body->>'direction'=$2 OR body->>'direction'='all') AND ($1='' OR to_tsvector('russian',body::text) @@ websearch_to_tsquery('russian',$1) OR body::text ILIKE '%' || $1 || '%') ORDER BY CASE WHEN $1='' THEN 0 ELSE ts_rank(setweight(to_tsvector('russian',body->>'title'),'A') || setweight(to_tsvector('russian',body->>'summary'),'B') || setweight(to_tsvector('russian',body::text),'D'),websearch_to_tsquery('russian',$1)) END DESC,id LIMIT 30`, query, direction)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Article{}
	for rows.Next() {
		var b []byte
		var x Article
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
func (a *App) getArticle(w http.ResponseWriter, r *http.Request) error {
	x, e := a.article(r.Context(), r.PathValue("id"))
	if e != nil {
		return e
	}
	var n int
	e = a.db.QueryRow(r.Context(), "SELECT count(*) FROM records WHERE kind='read' AND user_id=$1 AND body->>'article_id'=$2", current(r).ID, x.ID).Scan(&n)
	x.Read = n > 0
	if e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) readArticle(w http.ResponseWriter, r *http.Request) error {
	x, e := a.article(r.Context(), r.PathValue("id"))
	if e != nil {
		return e
	}
	u := current(r)
	if e = a.save(r.Context(), "read", u.ID, "read-"+u.ID+"-"+x.ID, map[string]string{"article_id": x.ID}); e != nil {
		return e
	}
	return respond(w, map[string]bool{"ok": true})
}
func (a *App) lessons(w http.ResponseWriter, r *http.Request) error {
	all, e := catalog[Lesson](r.Context(), a, "lesson")
	if e != nil {
		return e
	}
	attempts, e := records[Attempt](r.Context(), a, "attempt", current(r).ID)
	if e != nil {
		return e
	}
	out := []Lesson{}
	for _, l := range all {
		if l.Direction != current(r).Direction && l.Direction != "all" {
			continue
		}
		l.Questions = nil
		for _, at := range attempts {
			if at.LessonID == l.ID && at.Score >= 80 {
				l.Completed = true
			}
		}
		out = append(out, l)
	}
	return respond(w, out)
}
func (a *App) exam(w http.ResponseWriter, r *http.Request) error {
	l, e := a.lesson(r.Context(), r.PathValue("id"))
	if e != nil {
		return e
	}
	qs := []map[string]any{}
	for _, q := range l.Questions {
		qs = append(qs, map[string]any{"id": q.ID, "prompt": q.Prompt, "options": q.Options})
	}
	return respond(w, map[string]any{"id": l.ID, "title": l.Title, "questions": qs})
}
func (a *App) submitExam(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Answers map[string]int `json:"answers"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	l, e := a.lesson(r.Context(), r.PathValue("id"))
	if e != nil {
		return e
	}
	if current(r).Role != "intern" {
		return problem(403, "Экзамен доступен стажёру")
	}
	at, e := gradeExam(l.ID, l.Questions, in.Answers)
	if e != nil {
		return problem(400, e.Error())
	}
	tx, e := a.db.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	b, _ := json.Marshal(at)
	if _, e = tx.Exec(r.Context(), "INSERT INTO records(id,kind,user_id,body) VALUES($1,'attempt',$2,$3)", at.ID, current(r).ID, b); e != nil {
		return e
	}
	if at.Score >= 80 && l.Direction == current(r).Direction {
		_, e = tx.Exec(r.Context(), `UPDATE records SET body=jsonb_set(body,'{status}','"completed"') WHERE user_id=$1 AND kind='task' AND body->>'kind'='exam'`, current(r).ID)
		if e != nil {
			return e
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return respond(w, at)
}
func (a *App) interns(ctx context.Context, actor User) ([]User, error) {
	if actor.Role == "intern" {
		return nil, problem(403, "Доступ только для наставника или руководителя")
	}
	query := `SELECT id FROM users WHERE role='intern' AND is_demo=$2 AND `
	if actor.Role == "mentor" {
		query += "mentor_id=$1"
	} else {
		query += "team_id=$1"
	}
	scope := actor.ID
	if actor.Role == "manager" {
		scope = actor.TeamID
	}
	rows, e := a.db.Query(ctx, query+" ORDER BY created_at", scope, actor.IsDemo)
	if e != nil {
		return nil, e
	}
	ids := []string{}
	for rows.Next() {
		var s string
		if e = rows.Scan(&s); e != nil {
			rows.Close()
			return nil, e
		}
		ids = append(ids, s)
	}
	rows.Close()
	if e = rows.Err(); e != nil {
		return nil, e
	}
	out := []User{}
	for _, uid := range ids {
		u, e := a.loadUser(ctx, uid)
		if e != nil {
			return nil, e
		}
		out = append(out, u)
	}
	return out, nil
}
func (a *App) progress(ctx context.Context, u User) (Intern, error) {
	x := Intern{ID: u.ID, Name: u.Name, TeamName: u.TeamName, WeakTopics: []string{}}
	tasks, e := records[Task](ctx, a, "task", u.ID)
	if e != nil {
		return x, e
	}
	x.Total = len(tasks)
	for _, t := range tasks {
		if t.Status == "completed" {
			x.Completed++
		}
	}
	if x.Total > 0 {
		x.Progress = x.Completed * 100 / x.Total
	}
	ats, e := records[Attempt](ctx, a, "attempt", u.ID)
	if e != nil {
		return x, e
	}
	latest := map[string]Attempt{}
	for _, at := range ats {
		latest[at.LessonID] = at
	}
	for _, at := range latest {
		for _, f := range at.Feedback {
			if !f.Correct {
				x.WeakTopics = append(x.WeakTopics, f.Question)
			}
		}
	}
	return x, nil
}
func (a *App) dashboard(ctx context.Context, u User) (map[string]any, error) {
	tasks, e := records[Task](ctx, a, "task", u.ID)
	if e != nil {
		return nil, e
	}
	meetings, e := records[Meeting](ctx, a, "meeting", u.ID)
	if e != nil {
		return nil, e
	}
	requests, e := a.visibleRequests(ctx, u)
	if e != nil {
		return nil, e
	}
	attempts, e := records[Attempt](ctx, a, "attempt", u.ID)
	if e != nil {
		return nil, e
	}
	own, e := a.progress(ctx, u)
	if e != nil {
		return nil, e
	}
	internList := []Intern{}
	stats := map[string]int{"interns": 0, "mentors": 0, "teams": 0, "average_progress": 0, "ai_messages": 0}
	teamStats := []map[string]any{}
	if u.Role != "intern" {
		users, e := a.interns(ctx, u)
		if e != nil {
			return nil, e
		}
		sum := 0
		for _, v := range users {
			p, e := a.progress(ctx, v)
			if e != nil {
				return nil, e
			}
			internList = append(internList, p)
			sum += p.Progress
		}
		stats["interns"] = len(users)
		if len(users) > 0 {
			stats["average_progress"] = sum / len(users)
		}
		stats["teams"] = 1
		var count int
		_ = a.db.QueryRow(ctx, "SELECT count(*) FROM users WHERE role='mentor' AND team_id=$1 AND is_demo=$2", u.TeamID, u.IsDemo).Scan(&count)
		stats["mentors"] = count
		_ = a.db.QueryRow(ctx, `SELECT count(*) FROM records r JOIN users x ON x.id=r.user_id WHERE r.kind='message' AND r.body->>'role'='user' AND x.team_id=$1 AND x.is_demo=$2 AND ($3='manager' OR x.mentor_id=$4 OR x.id=$4)`, u.TeamID, u.IsDemo, u.Role, u.ID).Scan(&count)
		stats["ai_messages"] = count
		teamStats = append(teamStats, map[string]any{"team_name": u.TeamName, "interns": len(users), "progress": stats["average_progress"]})
	}
	return map[string]any{"progress": own.Progress, "completed": own.Completed, "total": own.Total, "tasks": tasks, "meetings": meetings, "requests": requests, "attempts": attempts, "interns": internList, "stats": stats, "team_stats": teamStats}, nil
}
func (a *App) dashboardHandler(w http.ResponseWriter, r *http.Request) error {
	x, e := a.dashboard(r.Context(), current(r))
	if e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) listInterns(w http.ResponseWriter, r *http.Request) error {
	users, e := a.interns(r.Context(), current(r))
	if e != nil {
		return e
	}
	out := []Intern{}
	for _, u := range users {
		x, e := a.progress(r.Context(), u)
		if e != nil {
			return e
		}
		out = append(out, x)
	}
	return respond(w, out)
}
func (a *App) targetIntern(r *http.Request, modify bool) (User, error) {
	actor := current(r)
	u, e := a.loadUser(r.Context(), r.PathValue("id"))
	if e != nil {
		return u, e
	}
	if actor.Role == "intern" || !canView(actor, u) || actor.IsDemo != u.IsDemo || u.Role != "intern" {
		return u, problem(403, "Нет доступа к этому стажёру")
	}
	if modify && actor.Role != "mentor" {
		return u, problem(403, "Изменения доступны закреплённому наставнику")
	}
	return u, nil
}
func (a *App) internDashboard(w http.ResponseWriter, r *http.Request) error {
	u, e := a.targetIntern(r, false)
	if e != nil {
		return e
	}
	x, e := a.dashboard(r.Context(), u)
	if e != nil {
		return e
	}
	x["user"] = u
	return respond(w, x)
}
func (a *App) assignTask(w http.ResponseWriter, r *http.Request) error {
	u, e := a.targetIntern(r, true)
	if e != nil {
		return e
	}
	var in struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		DueAt       time.Time `json:"due_at"`
		ArticleID   string    `json:"article_id"`
	}
	if e = decode(r, &in); e != nil {
		return e
	}
	if strings.TrimSpace(in.Title) == "" || len(in.Title) > 300 || len(in.Description) > 5000 || in.DueAt.IsZero() {
		return problem(400, "Укажите название и срок задачи")
	}
	if in.ArticleID != "" {
		if _, e = a.article(r.Context(), in.ArticleID); e != nil {
			return problem(400, "Материал не найден")
		}
	}
	t := Task{ID: id(), Title: in.Title, Description: in.Description, DueAt: in.DueAt, ArticleID: in.ArticleID, Status: "not_started", Kind: "manual", Priority: "medium"}
	if e = a.save(r.Context(), "task", u.ID, t.ID, t); e != nil {
		return e
	}
	return respond(w, t)
}
func (a *App) visibleRequests(ctx context.Context, u User) ([]Request, error) {
	if u.Role == "intern" {
		return records[Request](ctx, a, "request", u.ID)
	}
	users, e := a.interns(ctx, u)
	if e != nil {
		return nil, e
	}
	out := []Request{}
	for _, v := range users {
		items, e := records[Request](ctx, a, "request", v.ID)
		if e != nil {
			return nil, e
		}
		out = append(out, items...)
	}
	return out, nil
}
func (a *App) listRequests(w http.ResponseWriter, r *http.Request) error {
	x, e := a.visibleRequests(r.Context(), current(r))
	if e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) newRequest(ctx context.Context, u User, resource, reason string, confirmed bool) (Request, error) {
	var x Request
	if u.Role != "intern" {
		return x, problem(403, "Заявки создаёт стажёр")
	}
	resource = strings.TrimSpace(resource)
	reason = strings.TrimSpace(reason)
	if !confirmed {
		return x, problem(400, "Нужно явное подтверждение заявки")
	}
	if resource == "" || reason == "" || len(resource) > 200 || len(reason) > 3000 {
		return x, problem(400, "Укажите ресурс и обоснование")
	}
	now := time.Now().UTC()
	x = Request{ID: id(), Resource: resource, Reason: reason, Status: "waiting_approval", UserName: u.Name, CreatedAt: now, History: []History{{"waiting_approval", now, "Демонстрационная заявка: ожидает решения закреплённого наставника. Реальные права не выдаются."}}}
	tx, e := a.db.Begin(ctx)
	if e != nil {
		return x, e
	}
	defer tx.Rollback(ctx)
	b, _ := json.Marshal(x)
	if _, e = tx.Exec(ctx, "INSERT INTO records(id,kind,user_id,body) VALUES($1,'request',$2,$3)", x.ID, u.ID, b); e != nil {
		return x, e
	}
	if repoResource(resource) {
		_, e = tx.Exec(ctx, `UPDATE records SET body=jsonb_set(body,'{status}','"waiting_approval"') WHERE kind='task' AND user_id=$1 AND body->>'kind'='access' AND body->>'article_id'='gitlab-access' AND body->>'status'<>'completed'`, u.ID)
		if e != nil {
			return x, e
		}
	}
	return x, tx.Commit(ctx)
}
func repoResource(resource string) bool {
	switch strings.ToLower(strings.TrimSpace(resource)) {
	case "gitlab", "backend-repository", "backend-service", "frontend-repository", "gitlab-repository":
		return true
	}
	return false
}

func (a *App) createRequest(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Resource  string `json:"resource"`
		Reason    string `json:"reason"`
		Confirmed bool   `json:"confirmed"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	x, e := a.newRequest(r.Context(), current(r), in.Resource, in.Reason, in.Confirmed)
	if e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) decision(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Approve *bool `json:"approve"`
	}
	if e := decode(r, &in); e != nil {
		return e
	}
	if in.Approve == nil {
		return problem(400, "Укажите решение")
	}
	actor := current(r)
	if actor.Role != "mentor" {
		return problem(403, "Согласование доступно наставнику")
	}
	tx, e := a.db.Begin(r.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(r.Context())
	var b []byte
	var owner string
	e = tx.QueryRow(r.Context(), "SELECT body,user_id FROM records WHERE id=$1 AND kind='request' FOR UPDATE", r.PathValue("id")).Scan(&b, &owner)
	if e != nil {
		return e
	}
	u, e := a.loadUser(r.Context(), owner)
	if e != nil {
		return e
	}
	if !canView(actor, u) || actor.IsDemo != u.IsDemo {
		return problem(403, "Это заявка чужого стажёра")
	}
	var x Request
	if e = json.Unmarshal(b, &x); e != nil {
		return e
	}
	if x.Status != "waiting_approval" {
		return problem(409, "По заявке уже принято решение")
	}
	x.Status = "rejected"
	note := "Наставник отклонил демонстрационную заявку"
	taskStatus := "blocked"
	if *in.Approve {
		x.Status = "approved"
		taskStatus = "completed"
		note = "Наставник согласовал. Mock API подтвердил учебный доступ; реальные права не выдавались."
	}
	x.History = append(x.History, History{x.Status, time.Now().UTC(), note})
	b, _ = json.Marshal(x)
	if _, e = tx.Exec(r.Context(), "UPDATE records SET body=$2 WHERE id=$1", x.ID, b); e != nil {
		return e
	}
	if repoResource(x.Resource) {
		if _, e = tx.Exec(r.Context(), "UPDATE records SET body=jsonb_set(body,'{status}',to_jsonb($2::text)) WHERE user_id=$1 AND kind='task' AND body->>'kind'='access' AND body->>'article_id'='gitlab-access' AND body->>'status'<>'completed'", owner, taskStatus); e != nil {
			return e
		}
	}
	if e = tx.Commit(r.Context()); e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) listMeetings(w http.ResponseWriter, r *http.Request) error {
	x, e := records[Meeting](r.Context(), a, "meeting", current(r).ID)
	if e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) demoMeeting(w http.ResponseWriter, r *http.Request) error {
	x := Meeting{ID: id(), Title: "Тестовое напоминание: Git и Code Review", StartsAt: time.Now().UTC().Add(2 * time.Minute), ArticleID: "git-review", NotificationStatus: "pending"}
	if e := a.save(r.Context(), "meeting", current(r).ID, x.ID, x); e != nil {
		return e
	}
	return respond(w, x)
}
func (a *App) assignMeeting(w http.ResponseWriter, r *http.Request) error {
	u, e := a.targetIntern(r, true)
	if e != nil {
		return e
	}
	var x Meeting
	if e = decode(r, &x); e != nil {
		return e
	}
	if strings.TrimSpace(x.Title) == "" || len(x.Title) > 300 || x.StartsAt.Before(time.Now()) || x.ArticleID == "" {
		return problem(400, "Укажите название, будущее время и учебный материал")
	}
	if _, e = a.article(r.Context(), x.ArticleID); e != nil {
		return problem(400, "Материал не найден")
	}
	x.ID = id()
	x.NotificationStatus = "pending"
	if e = a.save(r.Context(), "meeting", u.ID, x.ID, x); e != nil {
		return e
	}
	return respond(w, x)
}
