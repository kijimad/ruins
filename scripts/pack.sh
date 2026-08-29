#!/bin/bash

# Asepriteでスプライトシートをパッキングするスクリプト
# ※ファイル名が数字で終わると aseprite が JSON の filename が正しく出力されないので、画像のファイル名には末尾にアンダースコアを追加する

set -eu

cd "$(dirname "$0")"
cd ../assets/file/textures

# Asepriteでスプライトシートをパッキングする関数
# 引数:
#   $1: ソースディレクトリ（例: tiles, single, bg）
pack_sprites() {
	local source_dir="$1"
	local output_name="$1" # 入力ディレクトリと同じ名前を使う

	aseprite \
		--batch "./${source_dir}/"*.png \
		--sheet "./dist/${output_name}.png" \
		--data "./dist/${output_name}.json" \
		--sheet-type packed \
		--format json-array \
		--filename-format "{title}"
}

# メイン処理
main() {
	mkdir -p dist
	pack_sprites "tiles"
	pack_sprites "single"
	pack_sprites "bg"
}

main
