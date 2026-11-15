# API contract (JSON, same-origin cookies)
Errors: {error:string}. IDs are strings. JSON timestamps ISO8601. All list endpoints return arrays (not wrapped). Mutations use application/json. API prefix /api.

GET /health -> {status:"ok"}
GET /api/public/config -> {demo_mode:boolean,ai_mock_mode:boolean,telegram_username:string,app_url:string}
GET /api/teams -> [{id,name,direction}]
POST /api/auth/demo {role:"intern"|"mentor"|"manager",name?:string,team_id?:string} -> User; sets session. Demo mentor is shared seeded mentor so sees demo interns. Manager team scoped.
POST /api/auth/telegram {init_data:string} -> User
GET /api/auth/login-link -> {url:string,token:string}; POST /api/auth/exchange {token:string} -> User (pending 409); bot confirms token
POST /api/auth/logout -> {ok:true}
GET /api/me -> User {id,name,role,team_id,team_name,direction,mentor_name,notifications,reminder_minutes,telegram_id?,is_demo}
PATCH /api/me {notifications:boolean,reminder_minutes:number} -> User
DELETE /api/me -> {ok:true}
GET /api/dashboard -> {progress:number,completed:number,total:number,tasks:Task[],meetings:Meeting[],requests:Request[],attempts:Attempt[],interns:Intern[],stats:{interns:number,mentors:number,teams:number,average_progress:number,ai_messages:number},team_stats:[{team_name,interns,progress}]}
GET /api/tasks -> Task[] {id,title,description,status,kind,due_at,article_id,priority}; kind manual|access|exam
PATCH /api/tasks/:id {status:"in_progress"|"completed"} -> Task; only manual tasks mutable
GET /api/knowledge?q=&direction= -> Article[] {id,title,category,direction,summary,sections:[{id,title,content}],read:boolean,duration_minutes:number}
GET /api/knowledge/:id -> Article
POST /api/knowledge/:id/read -> {ok:true}
GET /api/lessons -> Lesson[] {id,title,description,direction,article_id,duration_minutes,completed:boolean}
GET /api/lessons/:id/exam -> {id,title,questions:[{id,prompt,options:string[]}]}
POST /api/lessons/:id/exam {answers:{[questionId]:number}} -> Attempt {id,lesson_id,score,correct,total,feedback:[{question,correct:boolean,answer,explanation}],recommendation,created_at}
GET /api/chat -> Message[] {id,role,content,sources?:[{title,url}],suggested_actions?:string[],created_at}
POST /api/chat {message:string,lesson_id?:string} -> Message
GET /api/requests -> Request[] {id,resource,reason,status,created_at,user_name,history:[{status,at,note}]}
POST /api/requests {resource:string,reason:string,confirmed:true} -> Request
POST /api/requests/:id/decision {approve:boolean} -> Request; mentor only assigned interns
GET /api/meetings -> Meeting[] {id,title,starts_at,article_id,notification_status}
POST /api/meetings/demo -> Meeting due in 2 minutes; own user
GET /api/interns -> Intern[] {id,name,team_name,progress,completed,total,weak_topics:string[]}
GET /api/interns/:id -> same shape as dashboard + user
POST /api/interns/:id/tasks {title,description,due_at,article_id?} -> Task
POST /api/interns/:id/meetings {title,starts_at,article_id} -> Meeting

Admin distinct auth:
POST /api/admin/login {password:string} -> {ok:true}
POST /api/admin/logout -> {ok:true}
GET /api/admin/settings -> {telegram_token_set:boolean,telegram_username:string,telegram_enabled:boolean,routerai_key_set:boolean,routerai_base_url:string,routerai_model:string,ai_mock_mode:boolean,app_url:string,ai_timeout_seconds:number,ai_max_tokens:number}
PUT /api/admin/settings same fields plus optional telegram_token,routerai_api_key; blank secret preserves existing. explicit clear_telegram_token / clear_routerai_key deletes. Validation rejects enabling integration without required fields.
POST /api/admin/test/telegram -> {ok:boolean,message:string,username?:string}
POST /api/admin/test/routerai -> {ok:boolean,message:string,models?:[{id:string}]}; tests configured model tool calling if selected
GET /api/admin/audit -> [{id,action,created_at}]
