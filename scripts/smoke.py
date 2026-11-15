#!/usr/bin/env python3
"""Public API acceptance walkthrough. All records are marked synthetic demo data."""
import os, json, urllib.request, urllib.error, http.cookiejar, uuid, sys, ssl
BASE=os.environ.get('APP_URL','https://ai-buddy.85.159.231.99.sslip.io').rstrip('/')
checks=[]
cleanup_clients=[]
class Client:
 def __init__(self):
  cafile = os.environ.get("SSL_CERT_FILE") or ("/etc/ssl/cert.pem" if ssl.get_default_verify_paths().cafile is None and os.path.exists("/etc/ssl/cert.pem") else None)
  self.opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()),urllib.request.HTTPSHandler(context=ssl.create_default_context(cafile=cafile)))
 def call(self,path,method='GET',data=None,expect=200,origin=BASE):
  body=json.dumps(data).encode() if data is not None else None
  req=urllib.request.Request(BASE+path,data=body,method=method,headers={'Content-Type':'application/json','Origin':origin})
  try:
   with self.opener.open(req,timeout=90) as r: status=r.status; value=json.load(r)
  except urllib.error.HTTPError as e:
   status=e.code
   try: value=json.load(e)
   except Exception: value={}
  assert status==expect, f'{method} {path}: expected {expect}, got {status}: {value}'
  return value

def check(label,fn):
 try: fn()
 except AssertionError as error: raise AssertionError(label+': '+str(error)) from error
 checks.append(label); print('PASS '+label,flush=True)

def main():
 anon=Client()
 check('health',lambda: assert_eq(anon.call('/health')['status'],'ok'))
 check('anonymous data protected',lambda: anon.call('/api/me',expect=401))
 check('anonymous admin settings protected',lambda: anon.call('/api/admin/settings',expect=401))
 cfg=anon.call('/api/public/config')
 assert cfg['demo_mode'] is True
 check('cross-origin mutations rejected',lambda: anon.call('/api/auth/demo','POST',{'role':'intern'},expect=403,origin='https://untrusted.example'))
 teams=anon.call('/api/teams'); assert len(teams)>=8
 team=next((t for t in teams if t['name'].lower()=='platform'),teams[0])
 intern,other,mentor,manager=Client(),Client(),Client(),Client()
 u=intern.call('/api/auth/demo','POST',{'role':'intern','name':'Smoke '+uuid.uuid4().hex[:8],'team_id':team['id']})
 cleanup_clients.append(intern)
 other.call('/api/auth/demo','POST',{'role':'intern','name':'Smoke other','team_id':team['id']})
 cleanup_clients.append(other)
 mentor.call('/api/auth/demo','POST',{'role':'mentor','team_id':team['id']})
 manager.call('/api/auth/demo','POST',{'role':'manager','team_id':team['id']})
 cleanup_clients.append(manager)
 check('admin role cannot be self-selected',lambda: anon.call('/api/auth/demo','POST',{'role':'admin','team_id':team['id']},expect=400))
 dash=intern.call('/api/dashboard'); before=dash['progress']; assert dash['total']>0
 tasks=intern.call('/api/tasks'); manual=next(t for t in tasks if t['kind']=='manual')
 check('cross-user task protected',lambda: other.call('/api/tasks/'+manual['id'],'PATCH',{'status':'completed'},expect=403))
 intern.call('/api/tasks/'+manual['id'],'PATCH',{'status':'completed'})
 check('manual task updates persisted progress',lambda: assert_gt(intern.call('/api/dashboard')['progress'],before))
 arts=intern.call('/api/knowledge?q=Git'); assert arts
 art=intern.call('/api/knowledge/'+arts[0]['id']); assert art['sections']
 check('knowledge read persisted',lambda: intern.call('/api/knowledge/'+art['id']+'/read','POST',{}))
 lessons=intern.call('/api/lessons'); assert lessons
 lesson=next((x for x in lessons if x['id']=='lesson-backend'),lessons[0])
 exam=intern.call('/api/lessons/'+lesson['id']+'/exam'); assert len(exam['questions'])>=5
 assert all('answer' not in q and 'correct' not in q for q in exam['questions'])
 known={'q1':1,'q2':0,'q3':1,'q4':2,'q5':0}
 answers={q['id']:known.get(q['id'],0) for q in exam['questions']}
 result=intern.call('/api/lessons/'+lesson['id']+'/exam','POST',{'answers':answers})
 assert result['score']==100 and result['total']>=5
 check('passing exam completes learning task',lambda: assert_true(any(t['kind']=='exam' and t['status']=='completed' for t in intern.call('/api/tasks'))))
 check('exam evaluated and persisted',lambda: assert_true(len(intern.call('/api/dashboard')['attempts'])>=1))
 chat=intern.call('/api/chat','POST',{'message':'Как получить доступ к репозиторию?'})
 assert chat['content'] and chat.get('sources')
 check('access question retrieves access instruction first',lambda: assert_true('/knowledge/gitlab-access#' in chat['sources'][0]['url']))
 long_chat=intern.call('/api/chat','POST',{'message':'Как получить доступ к репозиторию? '+('Подскажи порядок согласования. '*12)})
 check('long chat message accepted',lambda: assert_true(bool(long_chat['content'])))
 check('chat answer contains valid knowledge sources',lambda: assert_true(all(s['url'].startswith('/knowledge/') for s in chat['sources'])))
 check('request requires explicit confirmation',lambda: intern.call('/api/requests','POST',{'resource':'backend-service','reason':'Smoke acceptance','confirmed':False},expect=400))
 unrelated=intern.call('/api/requests','POST',{'resource':'demo-calendar','reason':'Smoke unrelated resource','confirmed':True})
 mentor.call('/api/requests/'+unrelated['id']+'/decision','POST',{'approve':True})
 check('unrelated resource does not complete GitLab task',lambda: assert_true(all(t['status']!='completed' for t in intern.call('/api/tasks') if t['kind']=='access')))
 request=intern.call('/api/requests','POST',{'resource':'backend-service','reason':'Smoke acceptance','confirmed':True})
 check('intern cannot approve access',lambda: intern.call('/api/requests/'+request['id']+'/decision','POST',{'approve':True},expect=403))
 check('other intern cannot inspect protected profile',lambda: other.call('/api/interns/'+u['id'],expect=403))
 own=mentor.call('/api/interns'); assert any(x['id']==u['id'] for x in own)
 mentor.call('/api/requests/'+request['id']+'/decision','POST',{'approve':True})
 done=intern.call('/api/requests'); approved=next(r for r in done if r['id']==request['id']); assert approved['status'] in ('completed','approved')
 check('mentor approval updates access task',lambda: assert_true(any(t['kind']=='access' and t['status']=='completed' for t in intern.call('/api/tasks'))))
 again=intern.call('/api/requests','POST',{'resource':'backend-service','reason':'Smoke rejected second request','confirmed':True})
 mentor.call('/api/requests/'+again['id']+'/decision','POST',{'approve':False})
 check('later rejection preserves completed access',lambda: assert_true(any(t['kind']=='access' and t['status']=='completed' for t in intern.call('/api/tasks'))))
 stats=manager.call('/api/dashboard'); assert stats['stats']['interns']>=1
 check('manager analytics use persisted records',lambda: assert_true(len(stats['team_stats'])>=1))
 intern.call('/api/me','PATCH',{'notifications':False,'reminder_minutes':30})
 check('notification preference persisted',lambda: assert_eq(intern.call('/api/me')['notifications'],False))
 meeting=intern.call('/api/meetings/demo','POST',{}); assert meeting['starts_at']
 check('demo meeting created',lambda: assert_true(len(intern.call('/api/meetings'))>=1))
 password=os.environ.get('ADMIN_PASSWORD')
 if password:
  admin=Client(); admin.call('/api/admin/login','POST',{'password':password})
  admin.call('/api/me',expect=401)
  settings=admin.call('/api/admin/settings')
  assert 'telegram_token' not in settings and 'routerai_api_key' not in settings
  check('admin secrets masked',lambda: assert_true('routerai_key_set' in settings))
  admin.call('/api/admin/settings','PUT',{'routerai_base_url':'http://127.0.0.1:8000'},expect=400)
  check('admin provider SSRF rejected',lambda: assert_true(True))
  admin.call('/api/admin/logout','POST',{})
  admin.call('/api/admin/settings',expect=401)
 intern.call('/api/me','DELETE',{})
 intern.call('/api/me',expect=401)
 other.call('/api/me','DELETE',{})
 manager.call('/api/me','DELETE',{})
 mentor.call('/api/auth/logout','POST',{})
 check('profile deletion invalidates session',lambda: assert_true(True))
 print(json.dumps({'passed':len(checks),'checks':checks},ensure_ascii=False,indent=2))
def assert_eq(a,b): assert a==b,(a,b)
def assert_gt(a,b): assert a>b,(a,b)
def assert_true(a): assert a
if __name__=='__main__':
 try: main()
 except Exception as e:
  print('FAIL '+str(e),file=sys.stderr); sys.exit(1)
 finally:
  for client in cleanup_clients:
   try: client.call('/api/me','DELETE',{})
   except Exception: pass
