#!/bin/bash

set -e

cd "$(dirname "${BASH_SOURCE[0]}")"

clear 2>/dev/null || true

CURRENT_DIR="$(pwd)"

if [ -n "$TERMUX_VERSION" ]; then
    if [ -z "$1" ]; then
        echo "No package specified"
        exit 1
    fi
    yes | termux-setup-storage &>/dev/null
    apt update
else
    echo "This script should run on Termux"
    exit 1
fi

yes | pkg install -y golang binutils termux-elf-cleaner

TMP_DIR="$(mktemp -d)"

cleanup(){
    _EXIT_CODE=$?
    rm -rf "$TMP_DIR"
    return $_EXIT_CODE
}

trap cleanup EXIT

cd "$TMP_DIR"

cp -rf "$CURRENT_DIR" src

cd src

./build_dynamic_bundle.sh "$@"

for cmd in $@; do
    echo "$(basename "$cmd")," >> available_commands.txt.tmp
done

sort available_commands.txt.tmp | uniq | tr '\n' ' ' | sed -E 's|, $||' > available_commands.txt

go mod tidy

go build -buildvcs=false -ldflags "-s -w -buildid=" -o run .

termux-elf-cleaner run &>/dev/null

strip -s -w run

rm -rf "$CURRENT_DIR/run"

mv run "$CURRENT_DIR"

cp -f "$CURRENT_DIR/run" /sdcard/run
