# aux4 computer linux info

## provider description

### should describe the provider with the contract fields

```execute
aux4 computer linux info | jq -c '{provider, contract, platform: (.platform | test("^linux/")), version: (.version | type), capabilities: (.capabilities | type), permissions: (.permissions | type)}'
```

```expect
{"provider":"linux","contract":1,"platform":true,"version":"string","capabilities":"object","permissions":"object"}
```

### should report capabilities as booleans

```execute
aux4 computer linux info | jq -c '[.capabilities[] | type] | unique'
```

```expect
["boolean"]
```

## session detection

### should report no graphical session without DISPLAY or WAYLAND_DISPLAY

```execute
env -u DISPLAY -u WAYLAND_DISPLAY aux4 computer linux info | jq -c '{session, screenshot: .capabilities.screenshot, input: .capabilities.input}'
```

```expect
{"session":"none","screenshot":false,"input":false}
```

### should report a Wayland session and what it can't do

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux info | jq -c '{session, unsupported: (.unsupported | index("screenshot") != null)}'
```

```expect
{"session":"wayland","unsupported":true}
```
