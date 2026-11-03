#### Description

Lists running apps: those with a normal window (from the window manager's client list) and those exposing an
accessibility tree. Each has `name` (the window class, else the accessibility name, else the process name), `pid`,
`class` (`WM_CLASS`), `process`, `active` (owns the active window), `hidden` (all windows minimized), `windows` (count)
and `accessibility` (whether it exposes an AT-SPI tree). Read-only.

#### Usage

```bash
aux4 computer linux apps
```

#### Example

```bash
aux4 computer linux apps
```

```json
{"apps":[{"accessibility":true,"active":false,"class":"Galculator","hidden":false,"name":"Galculator","pid":5685,"process":"galculator","windows":1},{"accessibility":true,"active":true,"class":"Mousepad","hidden":false,"name":"Mousepad","pid":5686,"process":"mousepad","windows":1}]}
```
