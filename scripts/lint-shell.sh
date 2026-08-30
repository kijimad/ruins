#!/bin/bash
set -eu

##################################
# シェルスクリプトをshellcheckで検査するスクリプト
#
# 対象はgit経由で集める。shfmtは.gitignoreを読まないため、ディレクトリを直接歩かせると
# Cataclysm-DDAやnode_modulesを巻き込む。shfmt -fはshebangで判別するので拡張子なしも拾う
##################################

cd "$(dirname "$0")"
cd ../

# 明示パスで呼ぶ。PATH上の別版に横取りされず、固定した版がそのまま効く
SHELLCHECK=.cache/bin/shellcheck
if [ ! -x "$SHELLCHECK" ]; then
	echo "❌ $SHELLCHECK が無い。make toolsinstall を実行する" >&2
	exit 1
fi

git ls-files -z | xargs -0 go tool shfmt -f | xargs -r "$SHELLCHECK"
