#!/bin/bash
# Steam アセットの最終合成。背景マスタを各カプセルサイズへ切り出し、ロゴを重ねて generated/ へ出す。
# パーツは各ディレクトリに同居する: background/(gen_master.py) と logo/(gen_logo.sh)。
# 使い方: nix-shell docs/steam/shell.nix 内で bash docs/steam/compose.sh  依存: ImageMagick

set -euo pipefail

# 一時ファイルは専用ディレクトリに集約し、終了時に自動で消す。/tmp 直下へのハードコードは
# 並列実行や複数端末で上書き競合するため使わない。
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# マスターは横1枚だけ。縦カプセルもこの1枚をポートレートに切り出す。
MASTER="${STEAM_MASTER:-docs/steam/background/master_3840x2560.png}"
OUT="${STEAM_OUT:-docs/steam/generated}"
# 完成品ロゴ。副題と、文字の後ろのぼかした影を含む。加工せずそのまま重ねる
LOGO_PNG="docs/steam/logo/logo.png"

# 入力が無ければ magick のエラーを待たず、先に何を実行すべきか示して止める
[ -f "$MASTER" ] || {
	echo "ERROR: $MASTER が無い。先に background/gen_master.py を実行する" >&2
	exit 1
}
[ -f "$LOGO_PNG" ] || {
	echo "ERROR: $LOGO_PNG が無い。先に logo/gen_logo.sh を実行する" >&2
	exit 1
}

mkdir -p "$OUT"

# --- 大きい画像 ---

# 炎はマスターの左下で下端に接している。横長はどれも下端を基準に切り出し、炎の根元を残す。

# Library Hero 3840x1240 横長パノラマ、下端から切り出し
magick "$MASTER" -gravity south -crop 3840x1240+0+0 +repage "$OUT/library_hero.png"
echo "library_hero.png (3840x1240)"

# Page Background 1438x810 下端クロップ → リサイズ
# 比率 1438:810 = 1.776:1 → 3840x2162 からクロップ
magick "$MASTER" -gravity south -crop 3840x2162+0+0 +repage -resize 1438x810! "$OUT/page_background.png"
echo "page_background.png (1438x810)"

# --- ロゴの拡縮 ---
# ロゴはぼかした影のために文字の周りへ透明な余白を持つ。画像全体を箱に収めると余白のぶん文字が小さくなるので、
# 文字と罫のある所、すなわち影より濃い所の大きさで倍率を決め、影ごと拡縮する
read -r LOGO_CW LOGO_CH LOGO_CX LOGO_CY <<<"$(magick "$LOGO_PNG" -alpha extract -threshold 90% -format '%@' info: | tr 'x+' '  ')"

# logo_scale は文字のある所が幅 max_w と高さ max_h の箱に収まる倍率を返す
logo_scale() {
	awk -v mw="$1" -v mh="$2" -v cw="$LOGO_CW" -v ch="$LOGO_CH" 'BEGIN { s = mw / cw; if (mh / ch < s) s = mh / ch; printf "%.6f", s }'
}

# --- ロゴ描画関数 ---
# 完成品ロゴを背景に重ねる。ぼかした影を含むので加工しない。
# 文字のある所が幅 max_w と 高さ logo_h の箱に収まるよう、アスペクトを保って拡縮してから合成する
render_logo() {
	local w=$1 h=$2 logo_h=$3 gravity=$4 y_off=$5 output=$6

	local max_w=$((w * 82 / 100))
	local scale
	scale=$(logo_scale "$max_w" "$logo_h")
	magick "$LOGO_PNG" -resize "$(awk -v s="$scale" 'BEGIN { printf "%.4f%%", s * 100 }')" "$TMP/logo.png"

	# 宛先を sRGB へ昇格してから合成する。透明ベース xc:none が Gray になり色が落ちるのを防ぐ。
	# 入力と出力を同一ファイルにせず中間へ書いてから置き換える。読み書き競合を避ける
	magick "$output" -colorspace sRGB \
		-gravity "$gravity" \
		"$TMP/logo.png" -geometry "+0+${y_off}" -compose Over -composite \
		"$TMP/merged.png"
	mv "$TMP/merged.png" "$output"
}

# --- ロゴ付きカプセル: マスタをクロップ・リサイズ + ロゴ ---

generate_capsule() {
	local w=$1 h=$2 fname=$3
	local crop_w=$4 crop_h=$5 logo_gravity=$6
	# クロップ基準と横オフセット。縦カプセルは炎が左下にあるため左寄りに切り出す
	local crop_gravity=${7:-center}
	local crop_xoff=${8:-0}

	magick "$MASTER" -gravity "$crop_gravity" -crop "${crop_w}x${crop_h}+${crop_xoff}+0" +repage \
		-resize "${w}x${h}!" "$OUT/$fname"

	# 横長画像はロゴを大きめ、縦長画像は控えめ
	# 小型画像ほどロゴ比率を上げて視認性を確保する
	local logo_h
	if [ "$w" -gt "$h" ]; then
		if [ "$h" -lt 300 ]; then
			logo_h=$((h / 2))
		else
			logo_h=$((h / 3))
		fi
	else
		logo_h=$((h / 4))
	fi
	# north の場合、高さの 1/10 をマージンとする
	local y_off=0
	if [ "$logo_gravity" = "north" ]; then
		y_off=$((h / 10))
	fi

	render_logo "$w" "$h" "$logo_h" "$logo_gravity" "$y_off" "$OUT/$fname"

	echo "$fname (${w}x${h})"
}

# 各カプセルのクロップサイズはマスターのアスペクト比に合わせて計算する。
# すべて横マスター1枚から切り出す。横長は下端を基準にし、縦長は左下の炎を左端から少し離して入れる。
# generate_capsule w h filename crop_w crop_h logo_gravity crop_gravity crop_xoff
generate_capsule 462 174 small_capsule.png 3840 1446 center south
generate_capsule 920 430 header_capsule.png 3840 1794 center south
generate_capsule 920 430 library_header.png 3840 1794 center south
generate_capsule 1232 706 main_capsule.png 3840 2200 center south
generate_capsule 748 896 vertical_capsule.png 2137 2560 north southwest 120
generate_capsule 600 900 library_capsule.png 1707 2560 north southwest 120

# --- ゲームタイトル用画像 960x720 ---
# シネマ配置。ロゴを左上、メニューは右下(ゲーム側で描画)に置く。炎はマスターの左下にあり、メニューと重ならない。
magick "$MASTER" -gravity southwest -crop 2560x1920+120+0 +repage -resize 960x720! "$TMP/title_bg.png"
TITLE_OUT="${STEAM_TITLE_OUT:-assets/file/textures/bg/title1_.png}"
# ロゴを左上へ。文字の幅は500px、文字の左上を (80, 88) に置く。メニューはゲームのタイトル画面が右下へ描く。
# ロゴはぼかした影を含む。影の余白のぶん画像の左上をずらして重ねる
TITLE_SCALE=$(logo_scale 500 720)
read -r TITLE_X TITLE_Y <<<"$(awk -v s="$TITLE_SCALE" -v cx="$LOGO_CX" -v cy="$LOGO_CY" 'BEGIN { printf "%d %d", 80 - cx * s, 88 - cy * s }')"
magick "$LOGO_PNG" -resize "$(awk -v s="$TITLE_SCALE" 'BEGIN { printf "%.4f%%", s * 100 }')" "$TMP/title_logo.png"
magick "$TMP/title_bg.png" "$TMP/title_logo.png" -gravity NorthWest -geometry "+${TITLE_X}+${TITLE_Y}" -compose over -composite "$TMP/title_merged.png"
mv "$TMP/title_merged.png" "$TITLE_OUT"
echo "title1_.png (960x720)"

# --- Library Logo: 透明背景 + ロゴ ---

magick -size 1280x720 xc:none "$OUT/library_logo.png"
render_logo 1280 720 180 center 0 "$OUT/library_logo.png"
echo "library_logo.png (1280x720)"

echo ""
echo "All assets generated in $OUT/"
