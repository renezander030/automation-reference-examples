#!/usr/bin/env python3
"""Limited Claude Code Bash tripwire, not a shell parser or a sandbox."""
import json,pathlib,re,sys

def blocked(payload):
    if not isinstance(payload,dict) or payload.get('tool_name')!='Bash': return 'invalid Bash hook envelope'
    tool=payload.get('tool_input');cwd=payload.get('cwd')
    if not isinstance(tool,dict) or not isinstance(tool.get('command'),str) or not tool['command'].strip(): return 'missing command'
    if not isinstance(cwd,str): return 'missing cwd'
    try:
        p=pathlib.Path(cwd)
        if not p.is_absolute() or '..' in p.parts or not p.is_dir() or str(p.resolve(strict=True))!=str(p): return 'cwd must be an existing canonical directory'
    except (OSError,ValueError): return 'invalid cwd'
    cmd=tool['command']
    dns=r'\b(dig|host|nslookup|drill|kdig|getent)\b'
    execute=r'\|\s*(sh|bash|zsh|dash|python\d*|node|perl|ruby)\b|\beval\b|\$\('
    if re.search(dns,cmd,re.I) and re.search(execute,cmd,re.I): return 'DNS output feeding an interpreter'
    # Deliberately conservative: every mention of these package CLIs is reviewed.
    # No environment variable, command prefix, or writable root list grants trust.
    if re.search(r'\b(npm|npx|yarn|pnpm|bun|pip\d*|uv)\b',cmd,re.I): return 'package command requires out-of-band human review'
    return None

def main():
    try: reason=blocked(json.load(sys.stdin))
    except (ValueError,TypeError,OSError): reason='malformed hook JSON'
    if reason:
        print('BLOCKED: '+reason,file=sys.stderr);return 2
    return 0
if __name__=='__main__': sys.exit(main())
