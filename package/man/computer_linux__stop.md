#### Description

Turns on the **stop switch** shared with `aux4 computer` and every provider: creates the `STOP` file in the state
directory (`${XDG_STATE_HOME:-~/.local/state}/aux4-computer`, or `AUX4_COMPUTER_STATE_DIR`). While it exists — or while
`AUX4_COMPUTER_STOP=1` is set — every command that sends input or changes state refuses with exit `1` and
`"stopped":true` before touching anything, dry runs included. Read-only commands keep working. Refusals are recorded
in the action log. `resume` (or `aux4 computer resume`) turns it off.

#### Usage

```bash
aux4 computer linux stop
```

#### Example

```bash
aux4 computer linux stop
```

```json
{"file":"/home/sally/.local/state/aux4-computer/STOP","stopped":true}
```

```bash
aux4 computer linux click --element 0.1 --app galculator
```

```json
{"command":"click","error":"Stopped: aux4 computer linux is not sending input or changing state (/home/sally/.local/state/aux4-computer/STOP exists). Run 'aux4 computer resume' (or 'aux4 computer linux resume') to allow it again.","stopped":true}
```
