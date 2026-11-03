# aux4 computer linux ocr

```timeout
60000
```

The fixture `fixtures/ocr-sample.png` is 600x200 and shows "Hello aux4" near the top left and "Total: 1,792" near the bottom right.

## recognizing an existing PNG

### should return every line of text in reading order

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png | jq -c '[.matches[].text]'
```

```expect
["Hello aux4","Total: 1,792"]
```

### should describe the source image

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png | jq -c '{source, width, height, scale, level, mode}'
```

```expect:json
{
  "source": "image",
  "width": 600,
  "height": 200,
  "scale": 1,
  "level": "accurate",
  "mode": "background"
}
```

### should give each line a frame and center

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png | jq '.matches[0] as $a | .matches[1] as $b | ($a.center.x > 100 and $a.center.x < 180 and $a.center.y > 35 and $a.center.y < 75) and ($b.center.x > 360 and $b.center.x < 450 and $b.center.y > 125 and $b.center.y < 170) and ($a.confidence > 0.3)'
```

```expect
true
```

### should work on Wayland, where it needs no screen capture

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux ocr --image fixtures/ocr-sample.png | jq -c '[.matches[].text]'
```

```expect
["Hello aux4","Total: 1,792"]
```

### should convert pixels to points with --scale

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --scale 2 | jq '.width == 300 and .height == 100 and .scale == 2 and .matches[0].center.x > 50 and .matches[0].center.x < 90'
```

```expect
true
```

## filtering with --text

### should only return lines containing the text, case-insensitively

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --text HELLO | jq -c '{text, matches: [.matches[].text]}'
```

```expect
{"text":"HELLO","matches":["Hello aux4"]}
```

### should ignore commas and whitespace, and locate just the matched text

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --text 1792 | jq '(.matches | length) == 1 and .matches[0].match.frame.width < .matches[0].frame.width and .matches[0].match.center.x > .matches[0].center.x'
```

```expect
true
```

### should return no matches, not an error, when the text isn't there

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --text nowhere | jq -c '.matches'
```

```expect
[]
```

## argument validation

### should reject an invalid level

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --level best
```

```error:partial
Invalid --level 'best'*?
```

### should not combine --image and --app

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --app mousepad
```

```error:partial
--image cannot be combined with --app*?
```

### should list the supported languages for an unknown one

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --languages xx-YY 2>&1 | jq -c '{unsupported: (.error | test("Unsupported --languages")), eng: (.supported | index("eng") != null)}'
```

```expect
{"unsupported":true,"eng":true}
```

### should accept BCP 47 codes

```execute
aux4 computer linux ocr --image fixtures/ocr-sample.png --languages en-US --text aux4 | jq '.matches | length'
```

```expect
1
```

### should explain that Wayland can't capture a window

```execute
env -u DISPLAY WAYLAND_DISPLAY=wayland-0 aux4 computer linux ocr --app mousepad 2>&1 | jq -c '{unsupported}'
```

```expect
{"unsupported":true}
```
