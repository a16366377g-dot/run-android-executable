#!/bin/bash

set -e

#cd "$(dirname "${BASH_SOURCE[0]}")"

clear 2>/dev/null || true

CURRENT_DIR="$(pwd)"

TMP_DIR="$(mktemp -d)"

if [ -n "$TERMUX_VERSION" ]; then
    apt update
else
    echo "This script should run on Termux"
    exit 1
fi

yes | pkg install -y ldd binutils command-not-found tur-repo root-repo x11-repo

cleanup(){
    _EXIT_CODE=$?
    rm -rf "$TMP_DIR"
    return $_EXIT_CODE
}

checkIsDynamic(){
    if readelf -d "$1" 2>/dev/null | grep -q "Dynamic section at"; then
        echo "$1"
    fi
}

trap cleanup EXIT

cd "$TMP_DIR"

if [ -n "$*" ]; then
    package_list=("$@")
else
    echo "Error, no package specified"
    exit 1
fi

getPkgPath(){
    unset package
    package="$1"
    if [[ "$package" == */* ]]; then
        checkIsDynamic "$(cd "$CURRENT_DIR"; readlink -f "$package")"
    else
        if ! command -v "$package" &>/dev/null; then
            pkg_install="$("$PREFIX/libexec/termux/command-not-found" "$package" 2>&1 | grep "pkg install" | head -n 1 | sed "s/.* //g")"
            if [ -n "$pkg_install" ]; then
                echo "" >&2
                yes | pkg install -y "$pkg_install" >&2
            fi
        fi
        checkIsDynamic "$(command -v "$package")"
    fi
}

for package in "${package_list[@]}"; do
    echo ""
    echo "  Selected package ${package}..."
    echo ""
    unset pkgPath
    pkgPath="$(getPkgPath "$package")"
    if [ -z "$pkgPath" ]; then
        echo -e "\n  Package $package not valid. Skipping..."
    else
        pkgBasename="$(basename "$pkgPath")"
        mkdir -p bin/lib
        cp -L "$pkgPath" "bin/$pkgBasename"
        echo -e "\n  Getting dependencies..."
        echo ""
        ldd "$pkgPath" | grep --line-buffered -F "/data/data/com.termux/" | sed --unbuffered "s/.* \//\//" | sed --unbuffered "s/ .*//" | tee shared_libs.txt
        for libpath in $(cat shared_libs.txt); do
            cp -L "$libpath" bin/lib &>/dev/null
        done
    fi
done

find bin -type d -exec chmod 755 "{}" \;
find bin -type f -exec chmod 700 "{}" \;

echo -e "\n Compressing packages..."
rm -rf res.tar.gz
tar -I "zstd -10" -cf res.tar.zst bin
echo -e "\n  Done\n\n-------\n"

rm -rf "$CURRENT_DIR/res.tar.zst"
mv res.tar.zst "$CURRENT_DIR"
