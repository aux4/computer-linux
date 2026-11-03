#### Description

Activates the app's front window (restoring it if minimized). Foreground by nature: `"mode":"foreground"`. Most commands
don't need it: they act on a background app with `--app`.

#### Usage

```bash
aux4 computer linux focus <app>
```

--app   App name, window class or pid

#### Example

```bash
aux4 computer linux focus galculator
```

```json
{"app":"Galculator","focused":"Galculator","mode":"foreground","success":true}
```
