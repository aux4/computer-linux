#!/usr/bin/env bash
# temporary: dump what the provider sees, to shape the live assertions
cd package
c() { echo "### $*"; timeout 60 aux4 computer linux "$@" 2>&1 | cut -c1-2500; echo; }
nohup galculator > /tmp/galculator.log 2>&1 &
nohup mousepad > /tmp/mousepad.log 2>&1 &
sleep 4
aux4 computer linux find "" --role AXButton --app galculator | jq -c '[.matches[] | {label, ref}]'
press() { REF=$(aux4 computer linux find "" --role AXButton --app galculator | jq -r --arg k "$1" '[.matches[] | select(.label == $k)][0].ref // empty'); echo "press $1 -> $REF"; [ -n "$REF" ] && aux4 computer linux click --element "$REF" --app galculator; }
for k in AC 2 5 6 '*' x × 7; do press "$k"; done
( c wait --app galculator --text 1792 --timeout 10s ) &
sleep 1.5; press "="; wait
c find 1792 --app galculator
c ocr --app galculator --text 1792
c wait --app galculator --text 1792 --source ocr --timeout 10s
c screenshot --app galculator --annotate true --output /tmp/shots/annotated.png
B=$(aux4 computer linux find "" --role AXButton --app galculator | jq '.matches|length'); echo "basic keys $B"
c menu --app galculator --path "View > Scientific Mode"
sleep 1
S=$(aux4 computer linux find "" --role AXButton --app galculator | jq '.matches|length'); echo "sci keys $S"
c menu --app galculator --path "View > Basic Mode"
c click --app galculator --x 100 --y 280
c type "Hello from aux4/computer-linux CI" --app mousepad
c select --app mousepad --all true
c clipboard copy --app mousepad
c clipboard read
c select --app mousepad --text aux4
c select --app mousepad --start 0 --length 5
c shortcut ctrl+a --app mousepad
c set --app mousepad --element 0.0.2.0.0.0 --value "Set by aux4/computer-linux CI"
c clipboard paste --text " pasted" --app mousepad
c find "" --role AXCheckBox --app mousepad
c find "" --role "combo box" --app mousepad
c window list mousepad
c window move --app mousepad --x 120 --y 140
c window resize --app mousepad --width 600 --height 400
c window list mousepad
c window minimize --app mousepad
c window restore --app mousepad
c hide mousepad
c apps
c unhide mousepad
c window fullscreen true --app mousepad
c window fullscreen false --app mousepad
c scroll --app mousepad --dy=-200
c dialog --app mousepad
c window close --app mousepad
c dialog --app mousepad
c state --app mousepad --depth 6
c shortcut ctrl+s --app mousepad --mode background
c open --file /etc/hostname --app mousepad --wait 10s
c quit mousepad --timeout 5s
c log --limit 50
c screenshot --output /tmp/shots/screen.png
c ocr --image /tmp/shots/screen.png --text galculator
c click --app galculator --x 10 --y 10 --count 2
c cursor
c move --x 10 --y 10
c drag --x 10 --y 10 --toX 50 --toY 50
c shortcut ctrl+shift+x
c focus galculator
c window raise --app galculator
c quit galculator
