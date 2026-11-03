# aux4 computer linux wait

## argument validation

### should need something to wait for

```execute
aux4 computer linux wait
```

```error:partial
Provide --text, --role or --app*?
```

### should reject an invalid source

```execute
aux4 computer linux wait hello --source eyes
```

```error:partial
Invalid --source 'eyes'*?
```

### should not combine --source ocr with --role

```execute
aux4 computer linux wait hello --source ocr --role AXButton
```

```error:partial
--source ocr reads pixels*?
```

### should reject an invalid timeout

```execute
aux4 computer linux wait --app mousepad --timeout soon
```

```error:partial
Invalid --timeout 'soon'*?
```

## timing out

### should say what it was waiting for and what it saw

```execute
aux4 computer linux wait --app ThisAppDoesNotExist12345 --timeout 500ms 2>&1 | jq -c '{error: (.error | test("^Timed out after 500ms waiting for a window in ThisAppDoesNotExist12345")), running: .seen.running, timeout}'
```

```expect
{"error":true,"running":false,"timeout":0.5}
```

### should exit 1 on timeout

```execute
aux4 computer linux wait --app ThisAppDoesNotExist12345 --timeout 300ms 2>/dev/null; echo "exit=$?"
```

```expect
exit=1
```
