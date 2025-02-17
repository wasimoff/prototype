#!/usr/bin/env python3
import pickle, sys, base64
data = sys.stdin.read()
raw = base64.b64decode(data)
obj = pickle.loads(raw)
print(obj)
