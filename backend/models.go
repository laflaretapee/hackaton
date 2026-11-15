package main

import "time"

type Team struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Direction string `json:"direction"`
}
type User struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Role            string `json:"role"`
	TeamID          string `json:"team_id"`
	TeamName        string `json:"team_name"`
	Direction       string `json:"direction"`
	MentorID        string `json:"-"`
	MentorName      string `json:"mentor_name"`
	Notifications   bool   `json:"notifications"`
	ReminderMinutes int    `json:"reminder_minutes"`
	TelegramID      int64  `json:"telegram_id,omitempty"`
	IsDemo          bool   `json:"is_demo"`
}
type Task struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Kind        string    `json:"kind"`
	DueAt       time.Time `json:"due_at"`
	ArticleID   string    `json:"article_id"`
	Priority    string    `json:"priority"`
}
type Section struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}
type Article struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Category        string    `json:"category"`
	Direction       string    `json:"direction"`
	Summary         string    `json:"summary"`
	Sections        []Section `json:"sections"`
	Read            bool      `json:"read"`
	DurationMinutes int       `json:"duration_minutes"`
}
type Lesson struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Direction       string     `json:"direction"`
	ArticleID       string     `json:"article_id"`
	DurationMinutes int        `json:"duration_minutes"`
	Completed       bool       `json:"completed"`
	Questions       []Question `json:"questions,omitempty"`
}
type Question struct {
	ID          string   `json:"id"`
	Prompt      string   `json:"prompt"`
	Options     []string `json:"options"`
	Correct     int      `json:"correct"`
	Explanation string   `json:"explanation"`
}
type Feedback struct {
	Question    string `json:"question"`
	Correct     bool   `json:"correct"`
	Answer      string `json:"answer"`
	Explanation string `json:"explanation"`
}
type Attempt struct {
	ID             string     `json:"id"`
	LessonID       string     `json:"lesson_id"`
	Score          int        `json:"score"`
	Correct        int        `json:"correct"`
	Total          int        `json:"total"`
	Feedback       []Feedback `json:"feedback"`
	Recommendation string     `json:"recommendation"`
	CreatedAt      time.Time  `json:"created_at"`
}
type Source struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}
type Message struct {
	ID               string    `json:"id"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	Sources          []Source  `json:"sources,omitempty"`
	SuggestedActions []string  `json:"suggested_actions,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}
type History struct {
	Status string    `json:"status"`
	At     time.Time `json:"at"`
	Note   string    `json:"note"`
}
type Request struct {
	ID        string    `json:"id"`
	Resource  string    `json:"resource"`
	Reason    string    `json:"reason"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UserName  string    `json:"user_name"`
	History   []History `json:"history"`
}
type Meeting struct {
	ID                 string    `json:"id"`
	Title              string    `json:"title"`
	StartsAt           time.Time `json:"starts_at"`
	ArticleID          string    `json:"article_id"`
	NotificationStatus string    `json:"notification_status"`
}
type Intern struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	TeamName   string   `json:"team_name"`
	Progress   int      `json:"progress"`
	Completed  int      `json:"completed"`
	Total      int      `json:"total"`
	WeakTopics []string `json:"weak_topics"`
}
type Settings struct {
	TelegramTokenSet bool   `json:"telegram_token_set"`
	TelegramUsername string `json:"telegram_username"`
	TelegramEnabled  bool   `json:"telegram_enabled"`
	RouterAIKeySet   bool   `json:"routerai_key_set"`
	RouterAIBaseURL  string `json:"routerai_base_url"`
	RouterAIModel    string `json:"routerai_model"`
	AIMockMode       bool   `json:"ai_mock_mode"`
	AppURL           string `json:"app_url"`
	AITimeoutSeconds int    `json:"ai_timeout_seconds"`
	AIMaxTokens      int    `json:"ai_max_tokens"`
}
