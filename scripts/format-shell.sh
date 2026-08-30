#!/bin/bash
set -eu

##################################
# シェルスクリプトをshfmtで整形するスクリプト
#
# 対象をgit経由で集める理由はlint-shell.shに書いてある
##################################

cd "$(dirname "$0")"
cd ../

git ls-files -z | xargs -0 go tool shfmt -f | xargs -r go tool shfmt -w
