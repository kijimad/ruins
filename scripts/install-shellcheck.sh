#!/bin/bash
set -eu

##################################
# 静的解析ツール shellcheck を版固定でインストールするスクリプト
#
# 本体はHaskell製でgo installに乗らないため、配布バイナリを直接取得する。
# 置き場は .cache/bin にする。build_steam.sh が Steam Runtime や Go の tarball を置くのと
# 同じ、外部から落としたビルド資材の場所になる。make lint はここを直接指すのでPATHに依らない
#
# 版を上げるときはVERSIONとSHA256を対で書き換える。両者は常に一致していなければならない。
# 行頭が "# shellcheck" のコメントは解析ディレクティブと解釈されて構文エラーになるので避ける
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

cd "$(dirname "$0")"
cd ../

BIN_DIR=.cache/bin

if [ "$("$BIN_DIR/shellcheck" --version 2>/dev/null | awk '$1 == "version:" { print "v" $2 }')" = "$VERSION" ]; then
	echo "✅ shellcheck $VERSION は導入済み"
	exit 0
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

echo "✅ $PWD/$BIN_DIR/shellcheck に導入した"
