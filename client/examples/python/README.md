# Python / Ruby WASI Example

Use the precompiled Python releases from [vmware-labs/webassembly-language-runtimes](https://github.com/vmware-labs/webassembly-language-runtimes/releases), e.g.:
[python/3.12.0+20231211-040d5a6](https://github.com/vmware-labs/webassembly-language-runtimes/releases/download/python%2F3.12.0%2B20231211-040d5a6/python-3.12.0.wasm)

```
../../client.sh upload python-3.12.0.wasm
../../client.sh run python.json
```

There's also Ruby releases, which should work, too:
[ruby/3.2.2+20230714-11be424](https://github.com/vmware-labs/webassembly-language-runtimes/releases/download/ruby%2F3.2.2%2B20230714-11be424/ruby-3.2.2.wasm)