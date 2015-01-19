#!/usr/bin/env python2
import os
import sys
import subprocess
import random
import tempfile
from io import open
sys.stdin.close()

def get_env_or_exit(k):
    v = os.environ.get(k)
    if v is None:
        sys.stdout.write("%s not set\n" % k)
        sys.exit(1)
    return v

GOPATH = get_env_or_exit('GOPATH') 
PGHOST = get_env_or_exit('PGHOST')

proc = subprocess.Popen(['go','install','github.com/hydrogen18/sillyquill'])
if proc.wait() != 0:
    sys.stdout.write("go install failed\n")
    sys.exit(1)  

db_name = 'sillyquill_%d' % random.randint(1,65535) 
sql = "CREATE DATABASE %s;" % db_name
proc = subprocess.Popen(['psql','-c',sql,'postgres'])
if proc.wait() != 0:
    sys.stdout.write("creating database failed\n")
    sys.exit(1) 
 
passed = True 
with open('schema.sql','rb') as fin:
    proc = subprocess.Popen(['psql',db_name],stdin = fin)
    if proc.wait() != 0:
        sys.stdout.write("creating schema failed\n") 
        passed = False

if passed: 
    #TODO remove all '.go' files in 'dal'
    output_dir = os.path.join(GOPATH,'src','github.com','hydrogen18','sillyquill','gen_test','dal')
    with tempfile.NamedTemporaryFile(mode='wb',bufsize=0,suffix='sillyquill.toml',delete=True) as fout:
        fout.write('db="')
        fout.write('dbname=%s ' % db_name) 
        fout.write('sslmode=disable ')  
        fout.write('"\n')
        
        fout.write('schema="public"\n') 
        fout.write('package="dal"\n')
        
        fout.write('output-dir="')
        fout.write(output_dir)
        fout.write('"\n')
        exe = os.path.join(GOPATH,'bin','sillyquill')
        proc = subprocess.Popen([exe,'-conf',fout.name])
        retcode = proc.wait()
        sys.stdout.write("Sillyquill exited with code:%d\n" % retcode)
        if retcode != 0: 
            sys.stdout.write("Sillyquill failed\n") 
            passed = False

if passed:
    db_config = "dbname=%s sslmode=disable" % db_name
    proc_env = dict(os.environ)
    proc_env['DB'] = db_config
    proc = subprocess.Popen(['go','test','-v'],env=proc_env)  
    if proc.wait() != 0:
        sys.stdout.write("go test failed\n")
        passed = False
    
sql = "DROP DATABASE %s;" % db_name
proc = subprocess.Popen(['psql','-c',sql,'postgres'])
if proc.wait() != 0:
    sys.stdout.write("creating database failed\n")
    sys.exit(1) 
 
   
if passed:
    sys.exit(0)
else:
    sys.exit(1) 