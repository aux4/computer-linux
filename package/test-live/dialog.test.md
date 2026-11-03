# Live: dialog, close and quit (mousepad)

Drives mousepad's "unsaved changes" dialog: closing/quitting with unsaved text blocks on a
dialog, the dialog's shape and press, opening a file and waiting for its window, and a final
quit that is itself blocked by a second unsaved edit.

```timeout
25000
```

```beforeAll
aux4 computer linux open mousepad --wait 15s > /dev/null
```

```afterAll
aux4 computer linux quit mousepad --force true > /dev/null 2>&1 || true
```

## close with unsaved changes

### should type unsaved text and have dialog report none yet

```execute
aux4 computer linux type "Unsaved by aux4/computer-linux CI" --app mousepad > /dev/null
aux4 computer linux dialog --app mousepad | jq -c '.dialog'
```

```expect
null
```

### should block close with a dialog

```execute
aux4 computer linux window close --app mousepad | tee /tmp/computer-linux-close-unsaved.json | jq -c '{closed, hasDialog: (.dialog != null), noted: (.note | test("dialog --app"))}'
```

```expect
{"closed":false,"hasDialog":true,"noted":true}
```

### should report the dialog with multiple buttons

```execute
aux4 computer linux dialog --app mousepad | tee /tmp/computer-linux-dialog-open.json | jq -c '{hasKind: (.dialog.kind != null), buttons: (.dialog.buttons | length > 1)}'
```

```expect
{"hasKind":true,"buttons":true}
```

### should press Don't Save / Discard and close

```execute
BUTTON=$(jq -r '[.dialog.buttons[].title | select(test("^(Don.t Save|Close without Saving|Discard)"; "i"))][0] // empty' /tmp/computer-linux-dialog-open.json)
aux4 computer linux dialog --app mousepad --press "$BUTTON" --mode background | jq -c '{closed, mode}'
```

```expect
{"closed":true,"mode":"background"}
```

### should see mousepad close

```timeout
12000
```

```execute
for i in $(seq 1 50); do
  N=$(aux4 computer linux apps | jq '[.apps[] | select(.name == "Mousepad")] | length')
  [ "$N" -eq 0 ] && break
  sleep 0.2
done
aux4 computer linux apps | jq -c '[.apps[] | select(.name == "Mousepad")] | length'
```

```expect
0
```

## open --file --wait, then a second blocked quit

### should open a file and wait for its window

```timeout
17000
```

```execute
echo "Opened by aux4/computer-linux CI" > /tmp/computer-linux-open-test.txt
aux4 computer linux open --file /tmp/computer-linux-open-test.txt --app mousepad --wait 15s | jq -c '{hasTitle: (.window.title | test("computer-linux-open-test")), fast: (.waited < 15)}'
```

```expect
{"hasTitle":true,"fast":true}
```

### should find the opened file's content

```execute
aux4 computer linux find "Opened by aux4/computer-linux CI" --app mousepad | jq -c '.matches | length > 0'
```

```expect
true
```

### should block quit with a dialog after a further edit

```timeout
12000
```

```execute
aux4 computer linux type "more text " --app mousepad > /dev/null
aux4 computer linux quit mousepad --timeout 10s | tee /tmp/computer-linux-quit-asks.json | jq -c '{terminated, hasDialog: (.dialog != null), noted: (.note | test("dialog --app"))}'
```

```expect
{"terminated":false,"hasDialog":true,"noted":true}
```

### should press Don't Save / Discard and finish closing

```timeout
12000
```

```execute
BUTTON=$(jq -r '[.dialog.buttons[].title | select(test("^(Don.t Save|Close without Saving|Discard)"; "i"))][0] // empty' /tmp/computer-linux-quit-asks.json)
aux4 computer linux dialog --app mousepad --press "$BUTTON" | jq -c '.closed'
for i in $(seq 1 50); do
  N=$(aux4 computer linux apps | jq '[.apps[] | select(.name == "Mousepad")] | length')
  [ "$N" -eq 0 ] && break
  sleep 0.2
done
aux4 computer linux apps | jq -c '[.apps[] | select(.name == "Mousepad")] | length'
```

```expect:partial
true
0
```

## session apps

### should refuse to force-quit a session component (openbox)

```execute
aux4 computer linux quit openbox --force true
```

```error:partial
*?
```
