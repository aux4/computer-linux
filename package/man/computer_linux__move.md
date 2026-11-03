#### Description

Moves the mouse pointer to a screen point through XTEST, without clicking. Always real input (`"mode":"foreground"`).
Not available on Wayland.

#### Usage

```bash
aux4 computer linux move --x <x> --y <y>
```

--x   X in screen pixels
--y   Y in screen pixels

#### Example

```bash
aux4 computer linux move --x 500 --y 300
```

```json
{"mode":"foreground","x":500,"y":300}
```
