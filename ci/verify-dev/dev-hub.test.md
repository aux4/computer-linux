# Verify dev hub package (aux4/computer-linux)

Drives the package as installed from the DEV hub (not the working copy) against galculator on a
live X11 desktop: info/providers, background clicks through the provider and through the core's
OS dispatch, wait/OCR, quit, and the core's conformance suite. Run by
`.github/workflows/verify-dev.yml` via `aux4 test run --dir ci/verify-dev`, after the
Xvfb/openbox/dbus/AT-SPI desktop setup steps.

```timeout
20000
```

```afterAll
aux4 computer linux quit galculator --force true > /dev/null 2>&1 || true
```

## info and providers

### should report provider linux, contract 1, x11 session and the background/ocr capabilities

```execute
aux4 computer linux info | jq -c '{provider, contract, session, background: .capabilities.background, ocr: .capabilities.ocr}'
```

```expect
{"provider":"linux","contract":1,"session":"x11","background":true,"ocr":true}
```

### should match the pinned version when one is expected

```execute
VERSION=$(aux4 computer linux info | jq -r .version)
if [ -n "$EXPECTED_VERSION" ]; then [ "$VERSION" = "$EXPECTED_VERSION" ] && echo match || echo "mismatch: $VERSION != $EXPECTED_VERSION"; else echo "skip (no version pinned)"; fi
```

```expect:regex
^(match|skip \(no version pinned\))$
```

### should be seen by the core's providers list as ok

```execute
aux4 computer providers | jq -c '[.providers[] | select(.provider == "linux" and .status == "ok")] | length'
```

```expect
1
```

## 256 x 7 through the provider and the core's dispatch

### should open galculator

```execute
aux4 computer linux open galculator --wait 15s | jq -r .mode
```

```expect
background
```

### should press 7 in the background through the provider

```execute
REF=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r '[.matches[] | select(.label == "7")][0].ref')
aux4 computer linux click --element "$REF" --app galculator | jq -c '{mode, label: .element.label}'
```

```expect
{"mode":"background","label":"7"}
```

### should find 7 through the core's OS dispatch

```execute
aux4 computer find 7 --role text --app galculator | jq -c '[.matches[] | select(.value == "7")] | length > 0'
```

```expect
true
```

### should press AC 2 5 6 x 7 = through the core's dispatch

```execute
for k in AC 2 5 6 '*' 7 =; do
  R=$(aux4 computer find "" --role AXButton --app galculator | jq -r --arg k "$k" '[.matches[] | select(.label == $k)][0].ref')
  aux4 computer click --element "$R" --app galculator | jq -r '.mode'
done
```

```expect
background
background
background
background
background
background
background
```

## wait, OCR, quit and conformance

### should see 1792 via accessibility (wait)

```execute
aux4 computer wait --app galculator --text 1792 --timeout 10s | jq -c '.matches | length > 0'
```

```expect
true
```

### should see 1792 via OCR

```execute
aux4 computer ocr --app galculator --text 1792 | jq -c '.matches | length > 0'
```

```expect
true
```

### should quit galculator

```execute
aux4 computer linux quit galculator | jq -c '.terminated'
```

```expect
true
```

### should pass the core's conformance suite against the installed provider

```timeout
30000
```

```execute
aux4 computer conformance --provider linux 2>&1 | sed -E 's/\x1b\[[0-9;]*[a-zA-Z]//g' | grep -E '^Tests:'
```

```expect:regex
^Tests:\s+\d+ passed, \d+ total$
```
