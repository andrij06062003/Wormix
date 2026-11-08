"""Compile a decompiled 1.25.0 class into a SWF with FFDec, adding the imports the decompiler left out.

The 1.25.0 client is obfuscated and FFDec's decompiler drops some imports; FFDec then stops with
"X is not an existing type". This adds the import for X (found in the decompiled sources) and retries.

Usage: FFDEC=ffdec-cli.jar python3 fiximports.py <decompiled scripts dir> <class.as> <in.swf> <out.swf> <class name>
"""
import sys,os,re,subprocess
src,asfile,inswf,outswf,cls=sys.argv[1:6]
idx={}
for root,_,files in os.walk(src):
    for f in files:
        if f.endswith('.as'):
            pkg=os.path.relpath(root,src).replace(os.sep,'.')
            idx.setdefault(f[:-3],[]).append(pkg)
def dec(n):  # FFDec file names escape some characters (%22 = ", %3C = <, ...)
    import urllib.parse; return urllib.parse.unquote(n)
idx={dec(k):[dec(p) for p in v] for k,v in idx.items()}
for i in range(80):
    r=subprocess.run(['java','-jar',os.environ.get('FFDEC','ffdec-cli.jar'),'-replace',inswf,outswf,cls,asfile],capture_output=True,text=True)
    out=r.stdout+r.stderr
    m=re.search(r'SEVERE: (.+?) is not an existing type',out)
    if not m:
        print('done' if os.path.exists(outswf) else out[-1500:]); break
    name=m.group(1); pk=idx.get(name.strip('§')) or idx.get(name)
    if not pk: print('unknown',name); break
    s=open(asfile,encoding='utf-8').read()
    imp='   import %s.%s;\n'%(pk[0],name)
    if imp in s: print('already imported',name,pk); break
    s=s.replace('\n   import ','\n'+imp+'   import ',1)
    open(asfile,'w',encoding='utf-8').write(s); print('added',imp.strip())
