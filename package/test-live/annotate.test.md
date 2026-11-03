# Live: screenshot --annotate (galculator)

Checks the set-of-marks annotation: marks line up with accessibility elements, the image box
matches the window coordinates, --limit truncates, and clicking a mark's ref presses the right
button. Also exercises a plain full-screen screenshot.

```timeout
20000
```

```beforeAll
aux4 computer linux open galculator --wait 15s > /dev/null
```

```afterAll
aux4 computer linux quit galculator --force true > /dev/null 2>&1 || true
```

## annotated screenshot

### should annotate more than 10 elements with a positive width

```execute
mkdir -p /tmp/computer-linux-live-screenshots
aux4 computer linux screenshot --app galculator --annotate true --output /tmp/computer-linux-live-screenshots/annotated.png | tee /tmp/computer-linux-annotate.json | jq -c '{annotated, hasMarks: (.marks | length > 10), width: (.width > 0)}'
```

```expect
{"annotated":true,"hasMarks":true,"width":true}
```

### should have the 7 mark match the 7 button's accessibility frame

```execute
MARK=$(jq -c '[.marks[] | select(.label == "7")][0]' /tmp/computer-linux-annotate.json)
SEVEN=$(aux4 computer linux find "" --role AXButton --app galculator | jq -c '[.matches[] | select(.label == "7")][0]')
jq -n -c --argjson m "$MARK" --argjson e "$SEVEN" '($m.ref == $e.ref) and ($m.center.x >= $e.frame.x and $m.center.x <= ($e.frame.x + $e.frame.width) and $m.center.y >= $e.frame.y and $m.center.y <= ($e.frame.y + $e.frame.height))'
```

```expect
true
```

### should have the 7 mark's image box contain its windowCenter

```execute
MARK=$(jq -c '[.marks[] | select(.label == "7")][0]' /tmp/computer-linux-annotate.json)
echo "$MARK" | jq -c '.scale as $k | ((.windowCenter.x * $k) >= .image.x and (.windowCenter.x * $k) <= (.image.x + .image.width) and (.windowCenter.y * $k) >= .image.y and (.windowCenter.y * $k) <= (.image.y + .image.height))'
```

```expect
true
```

### should truncate to --limit 3 marks and report it in --note

```execute
aux4 computer linux screenshot --app galculator --annotate true --roles all --limit 3 --output /tmp/computer-linux-live-screenshots/limit.png | jq -c '{count: (.marks | length), truncated, noted: (.note | test("--limit 3"))}'
```

```expect
{"count":3,"truncated":true,"noted":true}
```

### should click the 7 mark's ref and press the 7 button

```execute
R=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r '[.matches[] | select(.label == "AC")][0].ref')
aux4 computer linux click --element "$R" --app galculator > /dev/null
sleep 0.3
MARK=$(jq -c '[.marks[] | select(.label == "7")][0]' /tmp/computer-linux-annotate.json)
aux4 computer linux click --element "$(echo "$MARK" | jq -r .ref)" --app galculator | jq -c '{mode, label: .element.label}'
```

```expect
{"mode":"background","label":"7"}
```

### should show 7 on the display after the click

```execute
sleep 0.3
aux4 computer linux find 7 --role text --app galculator | jq -c '[.matches[] | select(.value == "7")] | length > 0'
```

```expect
true
```

## full-screen screenshot

### should capture the whole 1280x800 display 0

```execute
aux4 computer linux screenshot --output /tmp/computer-linux-live-screenshots/screen.png | jq -c '{width, height, display}'
```

```expect
{"width":1280,"height":800,"display":0}
```
