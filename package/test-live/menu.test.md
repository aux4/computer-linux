# Live: menu (galculator)

Drives galculator's menu in the background: listing, switching View > Scientific/Basic Mode (and
checking it actually changed the key count), a missing item error, and a shortcut dispatched
through the app's menu (Edit > Copy Display Value).

```timeout
15000
```

```beforeAll
aux4 computer linux open galculator --wait 15s > /dev/null
```

```afterAll
aux4 computer linux quit galculator --force true > /dev/null 2>&1 || true
```

## listing and switching modes

### should list View in the menu

```execute
aux4 computer linux menu --app galculator | jq -c '[.menus[].title] | index("View") != null'
```

```expect
true
```

### should switch to Scientific Mode in the background and add keys

```execute
BASIC=$(aux4 computer linux find "" --role AXButton --app galculator | jq '.matches | length')
aux4 computer linux menu --app galculator --path "View > Scientific Mode" --mode background | jq -c '{mode, pressed, checked}'
sleep 1
SCI=$(aux4 computer linux find "" --role AXButton --app galculator | jq '.matches | length')
jq -n -c --argjson b "$BASIC" --argjson s "$SCI" '($s > $b)'
```

```expect:partial
{"mode":"background","pressed":"Scientific Mode","checked":true}
true
```

### should switch back to Basic Mode

```execute
BASIC_BEFORE=$(aux4 computer linux find "" --role AXButton --app galculator | jq '.matches | length')
aux4 computer linux menu --app galculator --path "view > basic mode" --mode background | jq -c '{path}'
sleep 1
BASIC_AFTER=$(aux4 computer linux find "" --role AXButton --app galculator | jq '.matches | length')
jq -n -c --argjson before "$BASIC_BEFORE" --argjson after "$BASIC_AFTER" '($after < $before)'
```

```expect:partial
{"path":["View","Basic Mode"]}
true
```

### should reject a missing menu item and list the available ones

```execute
aux4 computer linux menu --app galculator --path "View > No Such Item"
```

```error:partial
Available: *?
```

## menu-dispatched shortcut

### should copy the display value through Edit > Copy Display Value

```execute
aux4 computer linux shortcut ctrl+c --app galculator --mode background | jq -c '{method, mode, isCopy: (.menuItem | test("Copy"))}'
```

```expect
{"method":"menu","mode":"background","isCopy":true}
```

### should have copied the display value to the clipboard

```execute
aux4 computer linux clipboard read | jq -c '.text | test("[0-9]")'
```

```expect
true
```

### should reject a shortcut with no matching menu item

```execute
aux4 computer linux shortcut ctrl+shift+F12 --app galculator --mode background
```

```error:partial
*?
```
