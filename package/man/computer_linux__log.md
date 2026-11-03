#### Description

Shows the most recent entries of the action log shared with `aux4 computer` and the other providers. Every command
that sends input or changes state appends one JSON line to `actions.jsonl` in the state directory
(`${XDG_STATE_HOME:-~/.local/state}/aux4-computer`, or `AUX4_COMPUTER_STATE_DIR`), whether it succeeded, failed or was
refused: `ts`, `provider` (`linux`), `command`, `app`, `args` (`--text`/`--value` cut to 16 characters plus their
length), `mode`, `result` (`ok`/`error`), `error`, `method`/`fallbackReason`, `ms`. Read-only commands and dry runs are
not recorded. The file is private (`0600`) and rotated at 10MB (`actions.jsonl.1` is read too). The output also says
whether the stop switch is on.

#### Usage

```bash
aux4 computer linux log [--limit 20]
```

--limit  Number of recent entries, oldest first (default: `20`)

#### Example

```bash
aux4 computer linux log --limit 1
```

```json
{"entries":[{"app":"Galculator","args":{"app":"galculator","button":"left","count":"1","element":"0.0.0.1.2.0.3","mode":"auto"},"command":"click","method":"Click","mode":"background","ms":7,"provider":"linux","result":"ok","ts":"2026-11-03T03:21:22.871Z"}],"path":"/home/sally/.local/state/aux4-computer/actions.jsonl","stopped":false}
```
