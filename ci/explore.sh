#!/usr/bin/env bash
# temporary: dump what the provider sees, to shape the live assertions
cd package
c() { echo "### aux4 computer linux $*"; timeout 60 aux4 computer linux "$@" 2>&1 | cut -c1-3000; echo; }
c info
c permissions
nohup galculator > /tmp/galculator.log 2>&1 &
nohup mousepad > /tmp/mousepad.log 2>&1 &
sleep 4
c apps
c windows
c state --app galculator
c find "" --role AXButton --app galculator
c menu --app galculator --depth 3
c find "" --app galculator
c screenshot --app galculator --output /tmp/shots/galculator.png
c ocr --app galculator
c state --app mousepad
c menu --app mousepad --depth 2
c type "Hello from aux4/computer-linux CI" --app mousepad
c find "Hello" --app mousepad
c window list mousepad
c screen
c cursor
c clipboard copy --text "clip test"
c clipboard read
