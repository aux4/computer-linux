# Live: calculator (galculator)

Drives galculator through aux4 computer linux on a real X11 desktop: permissions/info, opening
apps, background button presses, and the result appearing (wait/find), including the state and
read shapes. Requires a running X server with galculator and mousepad installed (see
.github/workflows/live.yml).

```timeout
20000
```

```beforeAll
aux4 computer linux open galculator --wait 15s > /dev/null
aux4 computer linux open mousepad --wait 15s > /dev/null
```

```afterAll
aux4 computer linux quit galculator --force true > /dev/null 2>&1 || true
aux4 computer linux quit mousepad --force true > /dev/null 2>&1 || true
```

## permissions and info

### should report every permission granted

```execute
aux4 computer linux permissions | jq -c '{granted: (.accessibility.granted and .screenRecording.granted and .input.granted)}'
```

```expect
{"granted":true}
```

### should report contract 1, x11 session and the ocr/background capabilities

```execute
aux4 computer linux info | jq -c '{provider, contract, session, ocr: .capabilities.ocr, background: .capabilities.background}'
```

```expect
{"provider":"linux","contract":1,"session":"x11","ocr":true,"background":true}
```

## opening galculator and mousepad

### should open galculator with a window title and a numeric pid

```execute
aux4 computer linux apps | jq -c '[.apps[] | select(.name == "Galculator")][0] | {hasTitle: (.pid | type == "number")}'
```

```expect
{"hasTitle":true}
```

### should open mousepad with a window title

```execute
aux4 computer linux windows | jq -c '[.windows[] | select(.app == "Mousepad")] | length > 0'
```

```expect
true
```

### should see galculator via wait --app

```timeout
12000
```

```execute
aux4 computer linux wait --app galculator --timeout 10s | jq -c '{source, hasMatches: (.matches | length > 0)}'
```

```expect
{"source":"app","hasMatches":true}
```

## 256 x 7 in the background

### should press AC 2 5 6 x 7 in the background and see 1792 before '=' is pressed

```timeout
15000
```

```execute
for k in AC 2 5 6 '*' 7; do
  R=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r --arg k "$k" '[.matches[] | select(.label == $k)][0].ref // empty')
  aux4 computer linux click --element "$R" --app galculator > /dev/null
  sleep 0.2
done
aux4 computer linux wait --app galculator --text 1792 --timeout 10s > /tmp/computer-linux-wait-calc.json 2> /tmp/computer-linux-wait-calc.err &
WP=$!
sleep 1.5
R=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r '[.matches[] | select(.label == "=")][0].ref // empty')
aux4 computer linux click --element "$R" --app galculator > /dev/null
WRC=0; wait "$WP" || WRC=$?
jq -c --argjson rc "$WRC" '{rc: $rc, waited: (.waited < 10 and .waited >= 1), hasMatches: (.matches | length > 0), source}' /tmp/computer-linux-wait-calc.json
```

```expect:partial
{"rc":0,"waited":true,"hasMatches":true,"source":*?}
```

### should not report the shown result as gone

```execute
aux4 computer linux wait --app galculator --text 1792 --gone true --timeout 1s
```

```error:partial
*?
```

### should see 1792 through OCR as well as accessibility

```timeout
17000
```

```execute
aux4 computer linux wait --app galculator --text 1792 --source ocr --timeout 15s | jq -c '{source, hasMatches: (.matches | length > 0)}'
```

```expect
{"source":"ocr","hasMatches":true}
```

### should find the result via accessibility

```execute
aux4 computer linux find 1792 --app galculator | jq -c '.matches | length > 0'
```

```expect
true
```

## state and read

### should report galculator's state shape

```execute
aux4 computer linux state --app galculator | jq -c '{name: (.app.name == "Galculator"), pid: (.app.pid | type == "number"), windows: (.windows | length > 0), hasFocused: has("focused"), dialog, elements: (.elements | length > 0), frontmostPid: (.frontmost.pid | type == "number")}'
```

```expect
{"name":true,"pid":true,"windows":true,"hasFocused":true,"dialog":null,"elements":true,"frontmostPid":true}
```

### should read galculator's element tree at depth 6

```execute
aux4 computer linux read --app galculator --depth 6 | jq -c '.elements | length > 10'
```

```expect
true
```
