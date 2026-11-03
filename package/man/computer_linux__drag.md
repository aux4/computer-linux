#### Description

Presses the left button at `--x`/`--y`, moves through intermediate points and releases at `--toX`/`--toY`, through
XTEST. Always real input (`"mode":"foreground"`). Not available on Wayland.

#### Usage

```bash
aux4 computer linux drag --x <x> --y <y> --toX <x> --toY <y>
```

--x     Start X
--y     Start Y
--toX   End X
--toY   End Y

#### Example

```bash
aux4 computer linux drag --x 100 --y 200 --toX 400 --toY 200
```

```json
{"from":{"x":100,"y":200},"mode":"foreground","to":{"x":400,"y":200}}
```
