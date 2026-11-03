#### Description

Minimizes every window of an app without activating anything, and waits until all are minimized. Hiding a hidden app
changes nothing (`changed: false`).

#### Usage

```bash
aux4 computer linux hide <app>
```

--app   App name, window class or pid

#### Example

```bash
aux4 computer linux hide mousepad
```

```json
{
  "app": "Mousepad",
  "before": {
    "hidden": false
  },
  "changed": true,
  "hidden": true,
  "method": "iconify",
  "mode": "background",
  "pid": 3757
}
```
