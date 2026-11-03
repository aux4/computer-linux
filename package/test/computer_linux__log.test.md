# aux4 computer linux log

```afterAll
rm -rf state
```

## argument validation

### should reject an invalid limit

```execute
aux4 computer linux log --limit 0
```

```error
{"error":"Invalid --limit '0': use a positive number"}
```

## recording actions

### should record a failed input command with its error

```execute
rm -rf state; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux click --x 10 --y 20 --app ThisAppDoesNotExist12345 2>/dev/null; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log --limit 5 | jq -c '.entries[] | {provider, command, app, args, result, error, ts: (.ts | test("^20"))}'
```

```expect
{"provider":"linux","command":"click","app":"ThisAppDoesNotExist12345","args":{"app":"ThisAppDoesNotExist12345","button":"left","count":"1","mode":"auto","x":"10","y":"20"},"result":"error","error":"App not running: ThisAppDoesNotExist12345","ts":true}
```

### should truncate typed text

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux type "a long and private sentence typed by an agent" --app ThisAppDoesNotExist12345 2>/dev/null; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log --limit 1 | jq -r '.entries[0].args.text'
```

```expect
a long and priva… (45 chars)
```

### should not record read-only commands

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux find Save --app ThisAppDoesNotExist12345 2>/dev/null; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log | jq -c '[.entries[].command]'
```

```expect
["click","type"]
```

### should not record dry runs

```execute
AUX4_COMPUTER_DRY_RUN=1 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux type hi --app mousepad > /dev/null; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log | jq '.entries | length'
```

```expect
2
```

### should keep the log private

```execute
ls -l state/actions.jsonl | cut -c1-10
```

```expect
-rw-------
```

## rotation

### should rotate the file when it grows past its maximum and keep reading both

```execute
rm -rf state; for i in 1 2 3 4; do AUX4_COMPUTER_LOG_MAX_BYTES=300 AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux focus ThisAppDoesNotExist12345 2>/dev/null; done; test -f state/actions.jsonl.1 && echo rotated; AUX4_COMPUTER_STATE_DIR=$PWD/state aux4 computer linux log --limit 2 | jq -c '[.entries[].command]'
```

```expect
rotated
["focus","focus"]
```

## empty log

### should return no entries when nothing was recorded

```execute
AUX4_COMPUTER_STATE_DIR=$PWD/empty-state aux4 computer linux log | jq -c '{entries, stopped}'
```

```expect
{"entries":[],"stopped":false}
```
