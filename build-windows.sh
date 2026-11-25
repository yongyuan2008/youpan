#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")"

if [[ -x /opt/homebrew/opt/llvm/bin/llvm-rc && -x /opt/homebrew/opt/llvm/bin/llvm-cvtres ]]; then
  (
    cd installer
    /opt/homebrew/opt/llvm/bin/llvm-rc /C 65001 /fo app.res resources.rc
    /opt/homebrew/opt/llvm/bin/llvm-cvtres /MACHINE:X64 /OUT:resources_windows_amd64.syso app.res
  )
  cp installer/resources_windows_amd64.syso windows/resources_windows_amd64.syso
fi

(
  cd windows
  go test ./...
  GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags='-H=windowsgui -s -w' -o ../installer/DaysCountdown.exe .
)

if [[ -x "$PWD/tools/makensis/3.13/bin/makensis" ]]; then
  export NSISDIR="$PWD/tools/makensis/3.13/share/nsis"
  nsis_binary="$PWD/tools/makensis/3.13/bin/makensis"
else
  nsis_binary="${MAKENSIS:-makensis}"
fi
(
  cd installer
  "$nsis_binary" -V2 setup.nsi
)
file installer/DaysCountdown.exe installer/DaysCountdown-Setup-x64.exe
