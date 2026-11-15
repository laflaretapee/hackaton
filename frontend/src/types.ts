export type User = {
  id: string;
  name: string;
  role: "intern" | "mentor" | "manager";
  team_id: string;
  team_name: string;
  direction: string;
  mentor_name: string;
  notifications: boolean;
  reminder_minutes: number;
  is_demo: boolean;
};
export type Team = { id: string; name: string; direction: string };
export type Task = {
  id: string;
  title: string;
  description: string;
  status: string;
  kind: string;
  due_at: string;
  article_id: string;
  priority: string;
};
export type Article = {
  id: string;
  title: string;
  category: string;
  direction: string;
  summary: string;
  sections: { id: string; title: string; content: string }[];
  read: boolean;
  duration_minutes: number;
};
export type Lesson = {
  id: string;
  title: string;
  description: string;
  direction: string;
  article_id: string;
  duration_minutes: number;
  completed: boolean;
};
export type Attempt = {
  id: string;
  lesson_id: string;
  score: number;
  correct: number;
  total: number;
  feedback: {
    question: string;
    correct: boolean;
    answer: string;
    explanation: string;
  }[];
  recommendation: string;
  created_at: string;
};
export type Exam = {
  id: string;
  title: string;
  questions: { id: string; prompt: string; options: string[] }[];
};
export type Message = {
  id: string;
  role: string;
  content: string;
  sources?: { title: string; url: string }[];
  suggested_actions?: string[];
  created_at: string;
};
export type RequestItem = {
  id: string;
  resource: string;
  reason: string;
  status: string;
  created_at: string;
  user_name: string;
  history: { status: string; at: string; note: string }[];
};
export type Meeting = {
  id: string;
  title: string;
  starts_at: string;
  article_id: string;
  notification_status: string;
};
export type Intern = {
  id: string;
  name: string;
  team_name: string;
  progress: number;
  completed: number;
  total: number;
  weak_topics: string[];
};
export type Dashboard = {
  progress: number;
  completed: number;
  total: number;
  tasks: Task[];
  meetings: Meeting[];
  requests: RequestItem[];
  attempts: Attempt[];
  interns: Intern[];
  stats: {
    interns: number;
    mentors: number;
    teams: number;
    average_progress: number;
    ai_messages: number;
  };
  team_stats: { team_name: string; interns: number; progress: number }[];
  user?: User;
};
export type Config = {
  demo_mode: boolean;
  ai_mock_mode: boolean;
  telegram_username: string;
  app_url: string;
};
export type Settings = {
  telegram_token_set: boolean;
  telegram_username: string;
  telegram_enabled: boolean;
  routerai_key_set: boolean;
  routerai_base_url: string;
  routerai_model: string;
  ai_mock_mode: boolean;
  app_url: string;
  ai_timeout_seconds: number;
  ai_max_tokens: number;
};
