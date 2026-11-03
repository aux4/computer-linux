#!/usr/bin/env bash
# Live UI test on a headless X11 desktop (ci/session.sh): galculator + mousepad driven through
# aux4 computer linux, mirroring aux4/computer-mac's live test. Run from package/.
set -euo pipefail
T="${RUNNER_TEMP:-/tmp}/live"; mkdir -p "$T" "$T/screenshots"
cl() { aux4 computer linux "$@"; }
check() { # check <name> <jq filter> < json
  local name="$1" filter="$2" json; json=$(cat); echo "$json" > "$T/$name.json"; echo "$name: $json" | cut -c1-700
  if ! echo "$json" | jq -e "$filter" > /dev/null; then echo "::error::$name: expected $filter"; exit 1; fi
}
fails() { # fails <name> <command...>: the command must exit non-zero with a JSON error
  local name="$1"; shift
  if "$@" > "$T/$name.out" 2> "$T/$name.err"; then echo "::error::$name: expected a failure, got $(cat "$T/$name.out")"; exit 1; fi
  echo "$name (expected failure): $(tail -1 "$T/$name.err" | cut -c1-500)"
  tail -1 "$T/$name.err" | jq -e '.error | type == "string"' > /dev/null
}

echo "::group::permissions and info"
cl permissions | check permissions '.accessibility.granted and .screenRecording.granted and .input.granted'
cl info | check info '.provider == "linux" and .contract == 1 and .session == "x11" and .capabilities.ocr and .capabilities.background'
echo "::endgroup::"

echo "::group::open apps"
cl open galculator --wait 15s | check open-galculator '.window.title != null and (.pid | type == "number")'
cl open mousepad --wait 15s | check open-mousepad '.window.title != null'
cl wait --app galculator --timeout 10s | check wait-app '.source == "app" and (.matches | length > 0)'
echo "::endgroup::"

echo "::group::calculator 256x7 in the background"
press() {
  local key="$1" ref
  ref=$(cl find "" --role AXButton --app galculator | jq -r --arg k "$key" '[.matches[] | select(.label == $k)][0].ref // empty')
  if [ -z "$ref" ]; then
    echo "::error::no galculator button '$key'. Buttons: $(cl find "" --role AXButton --app galculator | jq -c '[.matches[].label]')"; exit 1
  fi
  cl click --element "$ref" --app galculator | check "press-$(echo "$key" | tr -c 'a-zA-Z0-9' '_')" '.mode == "background"'
  sleep 0.2
}
for k in AC 2 5 6 '*' 7; do press "$k"; done
# wait must notice the result appear: started before "=" is pressed
cl wait --app galculator --text 1792 --timeout 10s > "$T/wait-calc.json" 2> "$T/wait-calc.err" &
WP=$!
sleep 1.5
press "="
WRC=0; wait "$WP" || WRC=$?
cat "$T/wait-calc.json" "$T/wait-calc.err"
if [ "$WRC" != "0" ] || ! jq -e '.waited < 10 and .waited >= 1 and (.matches | length > 0) and (.source == "ax" or .source == "ocr")' "$T/wait-calc.json" > /dev/null; then
  echo "::error::wait --text 1792 did not see the result before its timeout (exit $WRC)"; exit 1
fi
fails wait-gone cl wait --app galculator --text 1792 --gone true --timeout 1s
cl wait --app galculator --text 1792 --source ocr --timeout 15s | check wait-ocr '.source == "ocr" and (.matches | length > 0)'

RESULT=$(cl find 1792 --app galculator)
echo "$RESULT" | check find-result '.matches | length > 0'
cl ocr --app galculator --text 1792 | check ocr-result '.matches | length > 0'
# OCR and accessibility agree on where the result is: the OCR windowCenter lies in the element's frame
ELEMENT=$(echo "$RESULT" | jq -c '[.matches[] | select(.frame.width != null and .windowCenter != null)][0]')
jq -e --argjson el "$ELEMENT" '
  ($el.center.x - $el.windowCenter.x) as $ox | ($el.center.y - $el.windowCenter.y) as $oy |
  .matches[0].match.windowCenter as $c |
  $c.x >= ($el.frame.x - $ox) and $c.x <= ($el.frame.x - $ox + $el.frame.width) and
  $c.y >= ($el.frame.y - $oy) and $c.y <= ($el.frame.y - $oy + $el.frame.height)' "$T/ocr-result.json" > /dev/null \
  || { echo "::error::OCR windowCenter is outside the accessibility frame of the result ($ELEMENT)"; exit 1; }

# A point found on screen hits the right control: click the 7 key at a window position, in the background.
KEY=$(cl ocr --app galculator --text 7 | jq -c '[.matches[] | select(.text == "7")] | max_by(.windowCenter.y) | .windowCenter // empty')
if [ -z "$KEY" ]; then
  echo "::warning::OCR did not read the 7 key on its own; using the accessibility windowCenter"
  KEY=$(cl find "" --role AXButton --app galculator | jq -c '[.matches[] | select(.label == "7")][0].windowCenter')
fi
cl click --app galculator --x "$(echo "$KEY" | jq .x)" --y "$(echo "$KEY" | jq .y)" --mode background | check click-point-7 '.mode == "background" and .element.label == "7"'
echo "::endgroup::"

echo "::group::screenshot --annotate"
cl screenshot --app galculator --annotate true --output "$T/screenshots/annotated.png" | check annotate '.annotated == true and (.marks | length > 10) and .width > 0'
MARK=$(jq -c '[.marks[] | select(.label == "7")][0]' "$T/annotate.json")
SEVEN=$(cl find "" --role AXButton --app galculator | jq -c '[.matches[] | select(.label == "7")][0]')
jq -n -e --argjson m "$MARK" --argjson e "$SEVEN" '$m.ref == $e.ref and $m.center.x >= $e.frame.x and $m.center.x <= ($e.frame.x + $e.frame.width) and $m.center.y >= $e.frame.y and $m.center.y <= ($e.frame.y + $e.frame.height)' > /dev/null \
  || { echo "::error::mark 7 ($MARK) does not match the 7 button ($SEVEN)"; exit 1; }
jq -e --argjson m "$MARK" '.scale as $k | ($m.windowCenter.x * $k) >= $m.image.x and ($m.windowCenter.x * $k) <= ($m.image.x + $m.image.width) and ($m.windowCenter.y * $k) >= $m.image.y and ($m.windowCenter.y * $k) <= ($m.image.y + $m.image.height)' "$T/annotate.json" > /dev/null \
  || { echo "::error::mark 7's image box does not contain its windowCenter"; exit 1; }
cl screenshot --app galculator --annotate true --roles all --limit 3 --output "$T/screenshots/limit.png" | check annotate-limit '(.marks | length) == 3 and .truncated == true and (.note | test("--limit 3"))'
press AC
cl click --element "$(echo "$MARK" | jq -r .ref)" --app galculator | check click-mark '.mode == "background" and .element.label == "7"'
sleep 0.3
cl find 7 --role text --app galculator | check display-7 '[.matches[] | select(.value == "7")] | length > 0'
cl screenshot --output "$T/screenshots/screen.png" | check screenshot-screen '.width == 1280 and .height == 800 and .display == 0'
echo "::endgroup::"

echo "::group::menu"
cl menu --app galculator | check menu-list '[.menus[].title] | index("View") != null'
BASIC=$(cl find "" --role AXButton --app galculator | jq '.matches | length')
cl menu --app galculator --path "View > Scientific Mode" --mode background | check menu-scientific '.mode == "background" and .pressed == "Scientific Mode" and .checked == true'
sleep 1
SCI=$(cl find "" --role AXButton --app galculator | jq '.matches | length')
echo "galculator keys: basic $BASIC, scientific $SCI"
[ "$SCI" -gt "$BASIC" ] || { echo "::error::View > Scientific Mode did not add keys"; exit 1; }
cl menu --app galculator --path "view > basic mode" --mode background | check menu-basic '.path == ["View", "Basic Mode"]'
sleep 1
[ "$(cl find "" --role AXButton --app galculator | jq '.matches | length')" -eq "$BASIC" ] || { echo "::error::View > Basic Mode did not switch back"; exit 1; }
fails menu-missing cl menu --app galculator --path "View > No Such Item"
grep -q "Available: " "$T/menu-missing.err"
# a shortcut with --app runs the app's menu item for it, in the background: Edit > Copy Display Value
cl shortcut ctrl+c --app galculator --mode background | check shortcut-menu '.method == "menu" and (.menuItem | test("Copy")) and .mode == "background"'
sleep 0.3
cl clipboard read | check shortcut-copied '.text | test("7")'
fails shortcut-no-item cl shortcut ctrl+shift+F12 --app galculator --mode background
echo "::endgroup::"

echo "::group::type, select, clipboard, set in the editor"
TEXT="Hello from aux4/computer-linux CI"
cl wait --app mousepad --text "$TEXT" --timeout 10s > "$T/wait-type.json" 2> "$T/wait-type.err" &
WP=$!
sleep 1
cl type "$TEXT" --app mousepad | check type '.mode == "background" and .method == "accessibility" and .typed == 33'
WRC=0; wait "$WP" || WRC=$?
cat "$T/wait-type.json" "$T/wait-type.err"
[ "$WRC" = "0" ] && jq -e '.waited < 10 and (.matches | length > 0)' "$T/wait-type.json" > /dev/null || { echo "::error::wait did not see the typed text"; exit 1; }
cl find "$TEXT" --app mousepad | check find-typed '.matches | length > 0'
cl select --app mousepad --all true --mode background | check select-all ".mode == \"background\" and .selected == \"$TEXT\" and .range.start == 0"
cl clipboard copy --app mousepad --mode background | check copy-selection ".mode == \"background\" and .text == \"$TEXT\""
cl clipboard read | check clipboard-read ".text == \"$TEXT\""
cl select --app mousepad --text aux4 --mode background | check select-text '.selected == "aux4" and .range.start == 11 and .range.length == 4'
cl select --app mousepad --start 0 --length 5 --mode background | check select-range '.selected == "Hello"'
fails select-missing cl select --app mousepad --text "not in the text"
cl shortcut ctrl+a --app mousepad --mode background | check shortcut-select-all '.mode == "background" and .selected == 33'
AREA=$(cl find "" --role AXTextArea --app mousepad | jq -r '[.matches[] | select(.role == "text")][0].ref')
cl set --app mousepad --element "$AREA" --value "Set by aux4/computer-linux CI" --mode background | check set-value ".before == \"$TEXT\" and .after == \"Set by aux4/computer-linux CI\" and .mode == \"background\""
cl select --app mousepad --start 4 --length 2 | check select-by '.selected == "by"'
cl clipboard paste --text "BY" --app mousepad | check paste '.mode == "background" and .pasted == "BY"'
cl find "" --role AXTextArea --app mousepad | check find-pasted '[.matches[] | select(.role == "text")][0].value == "Set BY aux4/computer-linux CI"'
cl clipboard copy --text "plain text" | check copy-text '.text == "plain text"'
cl clipboard read | check clipboard-read-2 '.text == "plain text"'
# scroll: a long document, then scroll its view in the background
LINES=$(seq 1 300 | sed 's/^/line /')
cl set --app mousepad --element "$AREA" --value "$LINES" > /dev/null
cl scroll --app mousepad --dy=-400 --mode background | check scroll '.mode == "background" and .method == "accessibility" and .vertical > 0'
cl set --app mousepad --element "$AREA" --value "short again" > /dev/null
echo "::endgroup::"

echo "::group::state"
cl state --app galculator | check state-calc '.app.name == "Galculator" and (.app.pid | type == "number") and (.windows | length > 0) and has("focused") and .dialog == null and (.elements | length > 0) and (.frontmost.pid | type == "number")'
cl state --app mousepad | check state-editor '[.elements[] | select(.role == "text")] | length > 0'
cl read --app galculator --depth 6 | check read '.elements | length > 10'
echo "::endgroup::"

echo "::group::windows"
cl windows | check windows '[.windows[] | select(.app == "Mousepad")] | length > 0'
cl window list mousepad | check window-list '.windows | length == 1'
cl window move --app mousepad --x 120 --y 140 --mode background | check window-move '.mode == "background" and .after.frame.x == 120 and .after.frame.y == 140'
cl window list mousepad | check window-move-readback '.windows[0].frame | .x == 120 and .y == 140'
cl window resize --app mousepad --width 600 --height 400 --mode background | check window-resize '.after.frame.width == 600 and .after.frame.height == 400'
cl window list mousepad | check window-resize-readback '.windows[0].frame | .width == 600 and .height == 400'
cl window minimize --app mousepad | check window-minimize '.before.minimized == false and .after.minimized == true'
cl screenshot --app mousepad --output "$T/screenshots/minimized.png" | check screenshot-minimized '.mode == "foreground" and (.fallbackReason | test("minimized"))'
cl window minimize --app mousepad > /dev/null || true
cl window restore --app mousepad | check window-restore '.before.minimized == true and .after.minimized == false'
cl window fullscreen true --app mousepad | check window-fullscreen '.after.fullscreen == true'
cl window fullscreen false --app mousepad | check window-fullscreen-off '.after.fullscreen == false'
cl window raise --app mousepad | check window-raise '.mode == "foreground" and .frontmost == true'
fails raise-background cl window raise --app mousepad --mode background
cl focus galculator | check focus '.mode == "foreground" and .success == true'
cl hide mousepad | check hide '.hidden == true and .changed == true and .mode == "background"'
cl apps | check hide-readback '[.apps[] | select(.name == "Mousepad")][0].hidden == true'
cl hide mousepad | check hide-again '.changed == false'
cl unhide mousepad | check unhide '.hidden == false and .changed == true'
echo "::endgroup::"

echo "::group::dialog, close and quit"
cl type "Unsaved by aux4/computer-linux CI" --app mousepad > /dev/null
cl dialog --app mousepad | check dialog-none '.dialog == null'
cl window close --app mousepad | check close-unsaved '.closed == false and .dialog != null and (.note | test("dialog --app"))'
cl dialog --app mousepad | check dialog-open '.dialog.kind != null and (.dialog.buttons | length > 1)'
BUTTON=$(jq -r '[.dialog.buttons[].title | select(test("^(Don.t Save|Close without Saving|Discard)"; "i"))][0] // empty' "$T/dialog-open.json")
[ -n "$BUTTON" ] || { echo "::error::no Don't Save button: $(jq -c '[.dialog.buttons[].title]' "$T/dialog-open.json")"; exit 1; }
cl dialog --app mousepad --press "$BUTTON" --mode background | check dialog-press ".closed == true and .mode == \"background\""
for i in $(seq 1 50); do [ "$(cl apps | jq '[.apps[] | select(.name == "Mousepad")] | length')" -eq 0 ] && break; sleep 0.2; done
cl apps | check editor-gone '[.apps[] | select(.name == "Mousepad")] | length == 0'

# open a file and wait for its window
echo "Opened by aux4/computer-linux CI" > "$T/aux4-open-test.txt"
cl open --file "$T/aux4-open-test.txt" --app mousepad --wait 15s | check open-file '(.window.title | test("aux4-open-test")) and .waited < 15'
cl find "Opened by aux4/computer-linux CI" --app mousepad | check open-file-content '.matches | length > 0'
cl type "more text " --app mousepad > /dev/null
cl quit mousepad --timeout 10s | check quit-asks '.terminated == false and .dialog != null and (.note | test("dialog --app"))'
BUTTON=$(jq -r '[.dialog.buttons[].title | select(test("^(Don.t Save|Close without Saving|Discard)"; "i"))][0] // empty' "$T/quit-asks.json")
cl dialog --app mousepad --press "$BUTTON" | check quit-dont-save '.closed == true'
for i in $(seq 1 50); do [ "$(cl apps | jq '[.apps[] | select(.name == "Mousepad")] | length')" -eq 0 ] && break; sleep 0.2; done
cl apps | check quit-gone '[.apps[] | select(.name == "Mousepad")] | length == 0'
fails quit-session cl quit openbox --force true
echo "::endgroup::"

echo "::group::real input (foreground)"
cl move --x 300 --y 300 | check move '.mode == "foreground"'
cl cursor | check cursor '.x == 300 and .y == 300'
cl click --app galculator --x 160 --y 58 --count 2 | check double-click '.mode == "foreground" and (.fallbackReason | test("multi-click")) and .restored.cursor == true'
cl cursor | check cursor-restored '.x == 300 and .y == 300'
echo "::endgroup::"

echo "::group::stop switch and log"
cl log --limit 200 | check log '[.entries[].command] as $c | ($c | index("click")) and ($c | index("type")) and ($c | index("window move")) and ($c | index("menu")) and ($c | index("dialog")) and ($c | index("quit"))'
cl stop | check stop '.stopped == true'
fails stopped-click cl click --element 0.0 --app galculator
jq -e '.stopped == true' < <(tail -1 "$T/stopped-click.err") > /dev/null
cl find "" --role AXButton --app galculator | check stopped-read-only '.matches | length > 0'
cl resume | check resume '.stopped == false'
press AC
cl quit galculator | check quit-calc '.terminated == true'
echo "live UI test passed"
