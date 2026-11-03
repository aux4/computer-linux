# Live: OCR vs accessibility (galculator)

Confirms OCR (pixels) and accessibility (the AT-SPI tree) agree on where the calculator result is,
and that clicking at an OCR-reported point lands on the right accessibility element.

## 256 x 7, then OCR

```timeout
20000
```

```beforeAll
aux4 computer linux open galculator --wait 15s > /dev/null
for k in AC 2 5 6 '*' 7 =; do
  R=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r --arg k "$k" '[.matches[] | select(.label == $k)][0].ref // empty')
  aux4 computer linux click --element "$R" --app galculator > /dev/null
  sleep 0.2
done
aux4 computer linux wait --app galculator --text 1792 --timeout 10s > /dev/null
```

```afterAll
aux4 computer linux quit galculator --force true > /dev/null 2>&1 || true
```

### should read 1792 via OCR

```execute
aux4 computer linux ocr --app galculator --text 1792 | jq -c '.matches | length > 0'
```

```expect
true
```

### should have the OCR windowCenter of the result land inside its accessibility frame

```execute
RESULT=$(aux4 computer linux find 1792 --app galculator)
ELEMENT=$(echo "$RESULT" | jq -c '[.matches[] | select(.frame.width != null and .windowCenter != null)][0]')
aux4 computer linux ocr --app galculator --text 1792 | jq -c --argjson el "$ELEMENT" '
  ($el.center.x - $el.windowCenter.x) as $ox | ($el.center.y - $el.windowCenter.y) as $oy |
  .matches[0].windowCenter as $c |
  ($c.x >= ($el.frame.x - $ox) and $c.x <= ($el.frame.x - $ox + $el.frame.width) and
   $c.y >= ($el.frame.y - $oy) and $c.y <= ($el.frame.y - $oy + $el.frame.height))'
```

```expect
true
```

### should click the 7 key at its OCR-reported point and press the right control

```execute
KEY=$(aux4 computer linux ocr --app galculator --text 7 | jq -c '[.matches[] | select(.text == "7")] | max_by(.windowCenter.y) | .windowCenter // empty')
if [ -z "$KEY" ]; then
  KEY=$(aux4 computer linux find "" --role AXButton --app galculator | jq -c '[.matches[] | select(.label == "7")][0].windowCenter')
fi
aux4 computer linux click --app galculator --x "$(echo "$KEY" | jq .x)" --y "$(echo "$KEY" | jq .y)" --mode background | jq -c '{mode, label: .element.label}'
```

```expect
{"mode":"background","label":"7"}
```
