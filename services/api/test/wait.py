"""Wait until each URL answers HTTP (any status)."""
import sys, time, urllib.request, urllib.error
for url in sys.argv[1:]:
    for _ in range(120):
        try:
            urllib.request.urlopen(url + "/api/__ping", timeout=2)
            break
        except urllib.error.HTTPError:
            break
        except Exception:
            time.sleep(0.5)
    else:
        print("timeout waiting for", url); sys.exit(1)
