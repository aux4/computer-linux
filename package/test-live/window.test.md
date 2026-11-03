# Live: window and hide/unhide (mousepad)

Drives mousepad's window: listing, move/resize with read-back, minimize/restore, fullscreen,
raise (foreground-only), focus, and hide/unhide.

```timeout
15000
```

```beforeAll
aux4 computer linux open mousepad --wait 15s > /dev/null
```

```afterAll
aux4 computer linux quit mousepad --force true > /dev/null 2>&1 || true
```

## listing and geometry

### should list mousepad once in the window list

```execute
aux4 computer linux window list mousepad | jq -c '.windows | length'
```

```expect
1
```

### should move the window and read back the new position

```execute
aux4 computer linux window move --app mousepad --x 120 --y 140 --mode background | jq -c '{mode, x: .after.frame.x, y: .after.frame.y}'
sleep 0.3
aux4 computer linux window list mousepad | jq -c '.windows[0].frame | {x, y}'
```

```expect:partial
{"mode":"background","x":120,"y":140}
{"x":120,"y":140}
```

### should resize the window and read back the new size

```execute
aux4 computer linux window resize --app mousepad --width 600 --height 400 --mode background | jq -c '{width: .after.frame.width, height: .after.frame.height}'
sleep 0.3
aux4 computer linux window list mousepad | jq -c '.windows[0].frame | {width, height}'
```

```expect:partial
{"width":600,"height":400}
{"width":600,"height":400}
```

## minimize, restore, fullscreen

### should minimize the window

```execute
aux4 computer linux window minimize --app mousepad | jq -c '{before: .before.minimized, after: .after.minimized}'
```

```expect
{"before":false,"after":true}
```

### should report a foreground fallback when screenshotting a minimized window

```execute
aux4 computer linux screenshot --app mousepad --output /tmp/computer-linux-minimized.png | jq -c '{mode, minimized: (.fallbackReason | test("minimized"))}'
```

```expect
{"mode":"foreground","minimized":true}
```

### should restore the window

```execute
aux4 computer linux window minimize --app mousepad > /dev/null 2>&1 || true
aux4 computer linux window restore --app mousepad | jq -c '{before: .before.minimized, after: .after.minimized}'
```

```expect
{"before":true,"after":false}
```

### should enter and exit fullscreen

```execute
aux4 computer linux window fullscreen true --app mousepad | jq -c '.after.fullscreen'
aux4 computer linux window fullscreen false --app mousepad | jq -c '.after.fullscreen'
```

```expect:partial
true
false
```

## raise and focus

### should raise mousepad to the foreground

```execute
aux4 computer linux window raise --app mousepad | jq -c '{mode, frontmost}'
```

```expect
{"mode":"foreground","frontmost":true}
```

### should reject raising in the background

```execute
aux4 computer linux window raise --app mousepad --mode background
```

```error:partial
*?
```

### should focus mousepad

```execute
aux4 computer linux focus mousepad | jq -c '{mode, success}'
```

```expect
{"mode":"foreground","success":true}
```

## hide and unhide

### should hide mousepad

```execute
aux4 computer linux hide mousepad | jq -c '{hidden, changed, mode}'
```

```expect
{"hidden":true,"changed":true,"mode":"background"}
```

### should read hidden back from apps

```execute
aux4 computer linux apps | jq -c '[.apps[] | select(.name == "Mousepad")][0].hidden'
```

```expect
true
```

### should report no change hiding an already-hidden app

```execute
aux4 computer linux hide mousepad | jq -c '.changed'
```

```expect
false
```

### should unhide mousepad

```execute
aux4 computer linux unhide mousepad | jq -c '{hidden, changed}'
```

```expect
{"hidden":false,"changed":true}
```
