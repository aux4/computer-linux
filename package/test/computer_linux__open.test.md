# aux4 computer linux open

## argument validation

### should need an app, a file or a url

```execute
aux4 computer linux open
```

```error:partial
Provide an app, --file or --url*?
```

### should reject a missing file

```execute
aux4 computer linux open --file /no/such/file.txt
```

```error:partial
File not found*?
```

### should reject a url without a scheme

```execute
aux4 computer linux open --url aux4.io
```

```error:partial
Invalid --url 'aux4.io'*?
```

### should not combine --hidden with --mode foreground

```execute
aux4 computer linux open mousepad --hidden true --mode foreground
```

```error:partial
--hidden true can't be combined*?
```

## dry run

### should open in the background

```execute
AUX4_COMPUTER_DRY_RUN=1 aux4 computer linux open mousepad | jq -c '{opened, mode}'
```

```expect
{"opened":"mousepad","mode":"background"}
```
