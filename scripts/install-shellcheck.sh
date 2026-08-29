#!/bin/bash
set -eu

##################################
# shellcheckを版固定でインストールするスクリプト
#
# shellcheckはHaskell製でgo installに乗らないため、配布バイナリを直接取得する。
# 版を上げるときはVERSIONとSHA256を対で書き換える。両者は常に一致していなければならない
##################################

VERSION=v0.11.0

case "$(uname -s)/$(uname -m)" in
Linux/x86_64)
	PLATFORM=linux.x86_64
	SHA256=8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198
	;;
Linux/aarch64)
	PLATFORM=linux.aarch64
	SHA256=12b331c1d2db6b9eb13cfca64306b1b157a86eb69db83023e261eaa7e7c14588
	;;
*)
	echo "❌ shellcheckの配布物を用意していない環境: $(uname -s)/$(uname -m)" >&2
	exit 1
	;;
esac

INSTALLED=$(shellcheck --version 2>/dev/null | awk '$1 == "version:" { print "v" $2 }')
if [ "$INSTALLED" = "$VERSION" ]; then
	echo "✅ shellcheck $VERSION は導入済み"
	exit 0
fi

# golangci-lintと同じ場所に置く。go installの出力先なのでPATHの通り方が揃う
BIN_DIR=$(go env GOBIN)
if [ -z "$BIN_DIR" ]; then
	BIN_DIR="$(go env GOPATH)/bin"
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

ARCHIVE="$TMP_DIR/shellcheck.tar.xz"
URL="https://github.com/koalaman/shellcheck/releases/download/$VERSION/shellcheck-$VERSION.$PLATFORM.tar.xz"

echo "📦 shellcheck $VERSION を取得する: $URL"
curl -fsSL "$URL" -o "$ARCHIVE"

if ! echo "$SHA256  $ARCHIVE" | sha256sum --check --status; then
	echo "❌ 取得したアーカイブのsha256が期待値と一致しない" >&2
	exit 1
fi

tar -xJf "$ARCHIVE" -C "$TMP_DIR"
mkdir -p "$BIN_DIR"
install -m 0755 "$TMP_DIR/shellcheck-$VERSION/shellcheck" "$BIN_DIR/shellcheck"

echo "✅ $BIN_DIR/shellcheck に導入した"
