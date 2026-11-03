#### Description

Turns off the shared stop switch by removing the `STOP` file (`removed` says whether it existed). When
`AUX4_COMPUTER_STOP=1` is still set in the environment, the output keeps `stopped: true` and says so.

#### Usage

```bash
aux4 computer linux resume
```

#### Example

```bash
aux4 computer linux resume
```

```json
{"file":"/home/sally/.local/state/aux4-computer/STOP","removed":true,"stopped":false}
```
