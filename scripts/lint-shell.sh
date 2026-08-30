#!/bin/bash
set -eu

##################################
# シェルスクリプトをshellcheckで検査するスクリプト
#
# 対象はgitの追跡ファイルをshfmt -fに通して選別する。shebangで判別するので拡張子なしの
# hooksも拾え、未追跡のCataclysm-DDAやnode_modulesは外れる。shfmt自身は.gitignoreを
# 読まないので、ディレクトリを直接歩かせるとそれらを巻き込む
##################################

cd "$(dirname "$0")"
cd ../

# go installに乗らない配布バイナリなので.cache/binに置き、明示パスで呼ぶ。
# PATHを見ないので、システムに別版が入っていても横取りされない
SHELLCHECK=.cache/bin/shellcheck
if [ ! -x "$SHELLCHECK" ]; then
	echo "❌ $SHELLCHECK が無い。make toolsinstall を実行する" >&2
	exit 1
fi

git ls-files -z | xargs -0 go tool shfmt -f | xargs -r "$SHELLCHECK"
