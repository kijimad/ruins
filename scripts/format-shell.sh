#!/bin/bash
set -eu

##################################
# シェルスクリプトをshfmtで整形するスクリプト
#
# 対象の選別はlint-shell.shと同じ理由でgit経由にする
##################################

cd "$(dirname "$0")"
cd ../

git ls-files -z | xargs -0 go tool shfmt -f | xargs -r go tool shfmt -w
