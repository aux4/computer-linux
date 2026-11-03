# aux4 computer linux notes

## platform notes

### should print markdown notes for agents

```execute
aux4 computer linux notes | head -1
```

```expect
## Linux notes (aux4/computer-linux)
```

### should explain Wayland and AT-SPI

```execute
aux4 computer linux notes | grep -c -E 'Wayland|AT-SPI' | awk '{print ($1 > 3) ? "covered" : "missing"}'
```

```expect
covered
```
