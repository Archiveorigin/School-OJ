#!/usr/bin/env python3
"""End-to-end judge capacity probe. Only runs against a seeded, isolated project.

Run on the Docker host: python3 scripts/judge_capacity.py /opt/School-OJ-judge-qa-...
Results are written to the project's tmp/capacity directory. Never use production.
"""
import base64
import datetime as dt
import hashlib
import hmac
import json
from pathlib import Path
import statistics
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
import uuid

root = Path(sys.argv[1]).resolve()
def output(args, **kw):
    return subprocess.check_output(args, cwd=root, text=True, **kw).strip()
config = json.loads(output(['docker', 'compose', 'config', '--format', 'json']))
assert config['name'].startswith('school-oj-judge-qa-'), 'requires an isolated QA project'
assert config['services']['api']['environment']['SEED_DATA'] == 'true'
assert root.name.startswith('School-OJ-judge-qa-')
ports = config['services']['web']['ports']
assert len(ports) == 1 and ports[0]['host_ip'] == '127.0.0.1'
base = 'http://127.0.0.1:' + str(ports[0]['published']) + '/api/v1'
evidence = root / 'tmp/capacity'; evidence.mkdir(parents=True, exist_ok=True)
def pg(sql):
    return output(['docker', 'compose', 'exec', '-T', 'postgres', 'psql', '-U', 'oj', '-d', 'oj', '-v', 'ON_ERROR_STOP=1', '-Atq'], input=sql)
def request(method, path, token='', data=None):
    headers = {'Authorization': 'Bearer ' + token} if token else {}
    if data is not None: headers['Content-Type'] = 'application/json'
    req = urllib.request.Request(base + path, data=json.dumps(data).encode() if data is not None else None, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=15) as response:
        raw = response.read()
        return json.loads(raw) if raw else None
def token_for(user_id):
    user = json.loads(pg(f"SELECT row_to_json(u) FROM (SELECT id,email,role,perm_version,token_version FROM users WHERE id={int(user_id)}) u;"))
    encode = lambda b: base64.urlsafe_b64encode(b).rstrip(b'=')
    claims = {'user_id': user['id'], 'role': user['role'], 'sub': user['email'], 'perm_version': user['perm_version'], 'token_version': user['token_version'], 'iat': int(time.time()), 'exp': int(time.time()) + 3600}
    raw = encode(b'{"alg":"HS256","typ":"JWT"}') + b'.' + encode(json.dumps(claims).encode())
    return (raw + b'.' + encode(hmac.new(config['services']['api']['environment']['JWT_SECRET'].encode(), raw, hashlib.sha256).digest())).decode()
admin_id = int(pg("SELECT id FROM users WHERE email='admin@school.local';"))
teacher_id = int(pg("SELECT id FROM users WHERE email='teacher@school.local';"))
admin, teacher = token_for(admin_id), token_for(teacher_id)
run_id = uuid.uuid4().hex[:12]
students = []
for i in range(8):
    students.append(int(pg(f"INSERT INTO users(email,name,role,password_hash,email_verified,created_at,updated_at) SELECT 'capacity-{run_id}-{i}@example.test','压测{i}','student',password_hash,true,now(),now() FROM users WHERE email='student@school.local' RETURNING id;")))
tokens = [token_for(i) for i in students]
team = request('POST', '/teams', teacher, {'name': '隔离容量验收', 'slug': 'capacity-' + run_id, 'visibility': 'private', 'join_mode': 'open', 'contest_permission': 'all'})
for student in students:
    pg(f"INSERT INTO team_memberships(team_id,user_id,role,created_at,updated_at) VALUES({team['id']},{student},'member',now(),now());")
future = dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=1)
contest = request('POST', f"/teams/{team['id']}/contests", teacher, {'title': '完整提交链路容量验收', 'starts_at': future.isoformat(), 'duration_minutes': 120, 'scoring_rule': 'ioi', 'ranking_user_ids': students[:6], 'gold_award_percent': 16.666666, 'silver_award_percent': 0, 'bronze_award_percent': 0})
paper = f"/event-papers/contest/{contest['id']}"
def revision(): return request('GET', paper, teacher)['review']['review_revision']
problems = []
sections = {'description': '读取内存 MiB 与 CPU 毫秒数，执行负载后输出 ok。', 'input_format': '一行两个整数：内存 MiB 与 CPU 毫秒数。', 'output_format': '输出一行 ok。', 'samples': [{'input': '64 250', 'output': 'ok'}]}
for points, ms in [(10, 250), (20, 100), (20, 250)]:
    problem = request('POST', paper + '/problems', teacher, {'revision': revision(), 'title': f'{points} 点 / {ms} ms', 'statement_sections': sections, 'time_limit_ms': 1000, 'memory_limit_mb': 512, 'output_limit_kb': 64, 'checker': {'type': 'tokens'}, 'test_mode': 'replace', 'label': chr(65 + len(problems)), 'score': 100, 'cases': [{'name': str(i), 'input': f'64 {ms}\n', 'output': 'ok\n', 'weight': 1} for i in range(points)]})
    problems.append(problem)
for action, actor in [('submit', teacher), ('approve', admin)]:
    request('POST', paper + '/review', actor, {'action': action, 'revision': revision()})
pg(f"UPDATE team_contests SET starts_at=now()-interval '1 minute' WHERE id={contest['id']};")
native = r'''#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <time.h>
static double cpu(){struct timespec t;clock_gettime(CLOCK_PROCESS_CPUTIME_ID,&t);return t.tv_sec+t.tv_nsec/1e9;}
int main(){int mib,ms;if(scanf("%d%d",&mib,&ms)!=2)return 1;volatile unsigned char *p=(unsigned char*)malloc((size_t)mib*1024*1024);if(!p)return 2;for(size_t i=0;i<(size_t)mib*1024*1024;i+=4096)p[i]=1;double until=cpu()+ms/1000.0;volatile uint64_t x=1;while(cpu()<until){for(int i=0;i<10000;++i)x=x*1664525+1013904223;}puts("ok");free((void*)p);}'''
python = '''import sys,time
mib,ms=map(int,sys.stdin.read().split())
p=bytearray(mib*1024*1024)
until=time.process_time()+ms/1000
x=1
while time.process_time()<until:
 for _ in range(10000):x=(x*1664525+1013904223)&0xffffffff
print('ok')
'''
java = '''import java.util.*;import java.lang.management.*;
public class Main{public static void main(String[]args){Scanner s=new Scanner(System.in);int mib=s.nextInt(),ms=s.nextInt();byte[] p=new byte[mib*1024*1024];for(int i=0;i<p.length;i+=4096)p[i]=1;ThreadMXBean b=ManagementFactory.getThreadMXBean();long end=b.getCurrentThreadCpuTime()+ms*1000000L,x=1;while(b.getCurrentThreadCpuTime()<end){for(int i=0;i<10000;i++)x=x*1664525+1013904223;}if(x==0&&p[0]==2)throw new RuntimeException();System.out.println("ok");}}'''
sources = {'c': native, 'cpp': native, 'python': python, 'java': java}
languages = ['cpp', 'python', 'cpp', 'java', 'cpp', 'c', 'cpp', 'python', 'cpp', 'cpp']
stop = threading.Event(); samples = []; failures = []; phase = 'warmup'
def monitor():
    previous = None
    while not stop.is_set():
        try:
            cpu = list(map(int, Path('/proc/stat').read_text().splitlines()[0].split()[1:9]))
            available = int(next(line.split()[1] for line in Path('/proc/meminfo').read_text().splitlines() if line.startswith('MemAvailable:'))) / 1024
            sample = {'at': time.time(), 'phase': phase, 'available_mib': available}
            if previous:
                delta = [a-b for a,b in zip(cpu, previous)]
                sample['cpu_percent'] = 100 * (1 - (delta[3]+delta[4]) / max(1, sum(delta)))
            previous = cpu
            at = time.monotonic()
            with urllib.request.urlopen('http://127.0.0.1:23333/healthz', timeout=2) as response: assert response.status == 200
            sample['production_health_ms'] = (time.monotonic()-at)*1000
            samples.append(sample)
            if available < 2048: raise RuntimeError('available memory below 2 GiB')
        except Exception as exc:
            failures.append(str(exc)); stop.set()
        stop.wait(1)
thread = threading.Thread(target=monitor, daemon=True); thread.start()
results = []
def check_guard():
    if stop.is_set(): raise RuntimeError('resource/health guard: ' + repr(failures))
def run_stage(name, count, interval=0, problem_index=0):
    global phase
    phase = name; began = time.monotonic(); submitted = []; seen = {}; concurrency = 0
    def poll():
        nonlocal concurrency
        rows = json.loads(pg(f"SELECT coalesce(json_agg(q),'[]') FROM (SELECT id,status FROM submissions WHERE team_contest_id={contest['id']}) q;"))
        concurrency = max(concurrency, sum(row['status'] == 'running' for row in rows))
        for row in rows:
            if row['status'] not in ('queued', 'running'): seen.setdefault(row['id'], (row['status'], time.monotonic()))
    for i in range(count):
        while time.monotonic() < began + interval*i:
            check_guard(); poll(); time.sleep(min(.5, max(0, began+interval*i-time.monotonic())))
        check_guard()
        lang = languages[i % len(languages)]
        index = (1 if i % 4 == 3 else 0) if problem_index == -1 else problem_index
        sent = time.monotonic()
        sub = request('POST', f"/contests/{contest['id']}/submissions", tokens[i % len(tokens)], {'problem_id': problems[index]['id'], 'language': lang, 'source_code': sources[lang]})
        submitted.append({'id': sub['id'], 'language': lang, 'problem_index': index, 'sent': sent})
        poll()
    deadline = time.monotonic() + 300
    while any(job['id'] not in seen for job in submitted):
        check_guard(); assert time.monotonic() < deadline, 'queue failed to drain'
        poll(); time.sleep(.5)
    elapsed = max(seen[job['id']][1] for job in submitted) - began
    for job in submitted:
        status, completed = seen[job['id']]
        assert status == 'accepted', (job, status)
        job['latency_seconds'] = completed-job.pop('sent')
    latencies = sorted(job['latency_seconds'] for job in submitted)
    result = {'stage': name, 'jobs': count, 'elapsed_seconds': elapsed, 'completed_per_minute': count*60/elapsed, 'p50_seconds': statistics.median(latencies), 'p95_seconds': latencies[min(count-1, int(count*.95))], 'max_running': concurrency, 'submissions': submitted}
    results.append(result)
    (evidence/'results.json').write_text(json.dumps(results, indent=2))
    print(json.dumps({k:v for k,v in result.items() if k != 'submissions'}), flush=True)
try:
    run_stage('warmup', 10)
    run_stage('burst-10-points-250ms', 30)
    run_stage('sustained-20-per-minute', 100, interval=3, problem_index=-1)
    run_stage('burst-20-points-250ms', 12, problem_index=2)
    ranking = request('GET', f"/contests/{contest['id']}/ranking", teacher)
    assert {row['user_id'] for row in ranking['rows']} == set(students[:6]), 'ranking scope mismatch'
    assert all(stage['max_running'] <= 2 for stage in results)
    assert any(stage['max_running'] == 2 for stage in results), 'second worker did not judge'
    assert pg(f"SELECT count(*) FROM submissions WHERE team_contest_id={contest['id']} AND user_id IN ({students[6]},{students[7]}) AND status='accepted';") != '0'
    sustained = results[2]
    burst_met = results[1]['completed_per_minute'] >= 20
    drain_seconds = sustained['elapsed_seconds'] - sustained['jobs'] * 3
    # 20/min refers to the injection rate being kept up: the five-minute stage
    # must drain immediately (bounded tail), with interactive latency and both
    # consumers active. The completed-per-minute figure includes that tail.
    sustained_met = drain_seconds <= 30 and sustained['p95_seconds'] <= 15 and sustained['max_running'] == 2
    summary = {'passed': burst_met and sustained_met, 'target_burst_met': burst_met, 'target_sustained_met': sustained_met, 'sustained_arrival_per_minute': 20.0, 'sustained_completed_per_minute': round(sustained['completed_per_minute'], 1), 'sustained_drain_seconds': round(drain_seconds, 1), 'sustained_p50_seconds': round(sustained['p50_seconds'], 2), 'sustained_p95_seconds': round(sustained['p95_seconds'], 2), 'total_jobs': sum(r['jobs'] for r in results), 'contest_id': contest['id'], 'ranking_excludes_but_submission_allows': True, 'health_failures': failures, 'minimum_available_mib': min(s['available_mib'] for s in samples) if samples else None, 'max_cpu_percent': max((s.get('cpu_percent', 0) for s in samples), default=0)}
    (evidence/'summary.json').write_text(json.dumps(summary, indent=2))
    print(json.dumps(summary), flush=True)
    assert summary['passed'], 'completed submissions per minute below the 20/min target'
finally:
    stop.set(); thread.join(timeout=3)
    (evidence/'samples.json').write_text(json.dumps(samples))
