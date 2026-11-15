#!/usr/bin/env python3
"""Generate the checked-in OpenAPI contract; review against handler changes."""
import json
from pathlib import Path

def obj(properties,required=()):return {'type':'object','properties':properties,**({'required':list(required)} if required else {})}
def ref(name):return {'$ref':'#/components/schemas/'+name}
def array(item):return {'type':'array','items':item}
s={'type':'string'};b={'type':'boolean'};n={'type':'integer'};dt={'type':'string','format':'date-time'}
schemas={
'Error':obj({'error':s},['error']),
'User':obj({**{k:s for k in ['id','name','team_id','team_name','direction','mentor_name']},'role':{'type':'string','enum':['intern','mentor','manager']},'notifications':b,'reminder_minutes':n,'telegram_id':{'type':'integer','format':'int64'},'is_demo':b}),
'Team':obj({k:s for k in ['id','name','direction']}),
'Task':obj({**{k:s for k in ['id','title','description','article_id','priority']},'status':{'type':'string','enum':['not_started','in_progress','waiting_approval','completed','blocked']},'kind':{'type':'string','enum':['manual','access','exam']},'due_at':dt}),
'Section':obj({k:s for k in ['id','title','content']}),
'Article':obj({**{k:s for k in ['id','title','category','direction','summary']},'sections':array(ref('Section')),'read':b,'duration_minutes':n}),
'Lesson':obj({**{k:s for k in ['id','title','description','direction','article_id']},'duration_minutes':n,'completed':b}),
'Question':obj({'id':s,'prompt':s,'options':array(s)}),
'Exam':obj({'id':s,'title':s,'questions':array(ref('Question'))}),
'Attempt':obj({'id':s,'lesson_id':s,'score':n,'correct':n,'total':n,'created_at':dt,'recommendation':s,'feedback':array(obj({'question':s,'correct':b,'answer':s,'explanation':s}))}),
'Message':obj({'id':s,'role':s,'content':s,'created_at':dt,'sources':array(obj({'title':s,'url':s})),'suggested_actions':array(s)}),
'Request':obj({**{k:s for k in ['id','resource','reason','status','user_name']},'created_at':dt,'history':array(obj({'status':s,'at':dt,'note':s}))}),
'Meeting':obj({'id':s,'title':s,'starts_at':dt,'article_id':s,'notification_status':s}),
'Intern':obj({'id':s,'name':s,'team_name':s,'progress':n,'completed':n,'total':n,'weak_topics':array(s)}),
'Settings':obj({'telegram_token_set':b,'telegram_username':s,'telegram_enabled':b,'routerai_key_set':b,'routerai_base_url':{'type':'string','enum':['https://routerai.ru/api/v1']},'routerai_model':s,'ai_mock_mode':b,'app_url':s,'ai_timeout_seconds':{'type':'integer','minimum':5,'maximum':120},'ai_max_tokens':{'type':'integer','minimum':100,'maximum':8000}})
}
schemas['Dashboard']=obj({'progress':n,'completed':n,'total':n,'tasks':array(ref('Task')),'meetings':array(ref('Meeting')),'requests':array(ref('Request')),'attempts':array(ref('Attempt')),'interns':array(ref('Intern')),'stats':obj({k:n for k in ['interns','mentors','teams','average_progress','ai_messages']}),'team_stats':array(obj({'team_name':s,'interns':n,'progress':n}))})
paths={}
def endpoint(method,path,summary,response=None,body=None,auth='user',params=()):
 op={'summary':summary,'operationId':method+'_'+path.replace('/','_').replace('{','').replace('}',''),'security':[] if not auth else [{('adminCookie' if auth=='admin' else 'userCookie'):[]}],'responses':{'200':{'description':'Успешно','content':{'application/json':{'schema':response or obj({'ok':b})}}}}}
 for status,description in [('400','Ошибка валидации'),('401','Требуется вход'),('403','Недостаточно прав'),('404','Ресурс не найден'),('409','Конфликт состояния'),('429','Лимит запросов'),('500','Внутренняя ошибка'),('502','Ошибка провайдера'),('503','Не настроено или недоступно')]:
  op['responses'][status]={'description':description,'content':{'application/json':{'schema':ref('Error')}}}
 if body:op['requestBody']={'required':True,'content':{'application/json':{'schema':body}}}
 parameters=list(params)
 if '{id}' in path:parameters.append({'name':'id','in':'path','required':True,'schema':s})
 if parameters:op['parameters']=parameters
 paths.setdefault(path,{})[method]=op
endpoint('get','/health','Готовность API и PostgreSQL',obj({'status':s}),auth='')
endpoint('get','/api/public/config','Публичная конфигурация без секретов',obj({'demo_mode':b,'ai_mock_mode':b,'telegram_username':s,'app_url':s}),auth='')
endpoint('get','/api/teams','Демонстрационные команды',array(ref('Team')),auth='')
endpoint('post','/api/auth/demo','Синтетический демонстрационный вход',ref('User'),obj({'role':{'type':'string','enum':['intern','mentor','manager']},'name':s,'team_id':s},['role']),auth='')
endpoint('post','/api/auth/telegram','Проверка подписи Telegram initData',ref('User'),obj({'init_data':s},['init_data']),auth='')
endpoint('get','/api/auth/login-link','Одноразовая ссылка подтверждения ботом',obj({'url':s,'token':s}),auth='')
endpoint('post','/api/auth/exchange','Одноразовый обмен после подтверждения',ref('User'),obj({'token':s},['token']),auth='')
endpoint('post','/api/auth/logout','Завершение сессии',auth='')
endpoint('get','/api/me','Текущий пользователь',ref('User'))
endpoint('patch','/api/me','Настройки уведомлений',ref('User'),obj({'notifications':b,'reminder_minutes':{'type':'integer','minimum':1,'maximum':10080}}))
endpoint('delete','/api/me','Удаление демопрофиля; системный демонаставник только выходит')
endpoint('get','/api/dashboard','Персональный или командный обзор',ref('Dashboard'))
endpoint('get','/api/tasks','Собственные задачи',array(ref('Task')))
endpoint('patch','/api/tasks/{id}','Изменение своей ручной задачи',ref('Task'),obj({'status':{'type':'string','enum':['in_progress','completed']}},['status']))
endpoint('get','/api/knowledge','Полнотекстовый поиск статей',array(ref('Article')),params=[{'name':k,'in':'query','schema':s} for k in ['q','direction']])
endpoint('get','/api/knowledge/{id}','Статья с разделами',ref('Article'))
endpoint('post','/api/knowledge/{id}/read','Отметка об изучении')
endpoint('get','/api/lessons','Занятия по направлению',array(ref('Lesson')))
endpoint('get','/api/lessons/{id}/exam','Вопросы без ключей ответов',ref('Exam'))
endpoint('post','/api/lessons/{id}/exam','Проверка ответов, индексы вариантов с нуля',ref('Attempt'),obj({'answers':{'type':'object','additionalProperties':{'type':'integer','minimum':0}}},['answers']))
endpoint('get','/api/chat','Личная история AI-диалога',array(ref('Message')))
endpoint('post','/api/chat','Ответ AI или явно обозначенный деморежим',ref('Message'),obj({'message':{'type':'string','minLength':1,'maxLength':6000},'lesson_id':s},['message']))
endpoint('get','/api/requests','Заявки в разрешённой области',array(ref('Request')))
endpoint('post','/api/requests','Создание mock-заявки после подтверждения',ref('Request'),obj({'resource':s,'reason':s,'confirmed':{'type':'boolean','const':True}},['resource','reason','confirmed']))
endpoint('post','/api/requests/{id}/decision','Решение назначенного наставника',ref('Request'),obj({'approve':b},['approve']))
endpoint('get','/api/meetings','Собственные встречи',array(ref('Meeting')))
endpoint('post','/api/meetings/demo','Учебная встреча через две минуты',ref('Meeting'))
endpoint('get','/api/interns','Стажёры назначенного наставника или команды руководителя',array(ref('Intern')))
endpoint('get','/api/interns/{id}','Учебный профиль без личных AI-диалогов',obj({**schemas['Dashboard']['properties'],'user':ref('User')}))
endpoint('post','/api/interns/{id}/tasks','Назначение задачи наставником',ref('Task'),obj({'title':s,'description':s,'due_at':dt,'article_id':s},['title','due_at']))
endpoint('post','/api/interns/{id}/meetings','Назначение учебной встречи',ref('Meeting'),obj({'title':s,'starts_at':dt,'article_id':s},['title','starts_at','article_id']))
endpoint('post','/api/admin/login','Независимый вход администратора',body=obj({'password':{'type':'string','writeOnly':True}},['password']),auth='')
endpoint('post','/api/admin/logout','Завершение административной сессии',auth='admin')
endpoint('get','/api/admin/settings','Настройки без значений секретов',ref('Settings'),auth='admin')
settings_input=obj({**schemas['Settings']['properties'],'telegram_token':{'type':'string','writeOnly':True},'routerai_api_key':{'type':'string','writeOnly':True},'clear_telegram_token':b,'clear_routerai_key':b})
endpoint('put','/api/admin/settings','Полная замена настроек; пустой секрет сохраняет прежний',ref('Settings'),settings_input,auth='admin')
for service in ['telegram','routerai']:endpoint('post','/api/admin/test/'+service,'Проверка сохранённого подключения '+service,obj({'ok':b,'message':s,'username':s,'models':array(obj({'id':s}))}),auth='admin')
endpoint('get','/api/admin/audit','Административный аудит без секретов',array(obj({'id':s,'action':s,'created_at':dt})),auth='admin')
doc={'openapi':'3.1.0','info':{'title':'AI-buddy API','version':'1.0.0','description':'Хакатонный MVP. Только синтетические корпоративные данные; mock-доступы. Мутации application/json с проверкой Origin. Сессии устанавливаются через Set-Cookie; admin независим от публичного выбора роли.'},'servers':[{'url':'https://ai-buddy.85.159.231.99.sslip.io'},{'url':'http://localhost:8095'}],'paths':paths,'components':{'securitySchemes':{'userCookie':{'type':'apiKey','in':'cookie','name':'buddy_session'},'adminCookie':{'type':'apiKey','in':'cookie','name':'buddy_admin'}},'schemas':schemas}}
Path('docs/openapi.json').write_text(json.dumps(doc,ensure_ascii=False,indent=2)+'\n')
print(f'OpenAPI: {len(paths)} paths, {sum(len(v) for v in paths.values())} operations')
