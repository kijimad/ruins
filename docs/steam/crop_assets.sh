#!/bin/bash
# Steam アセット クロップスクリプト
# マスター画像 (3840x2560) から各アセットサイズを切り出す
#
# 使い方: bash docs/steam/crop_assets.sh
#
# マスター画像の再生成:
#   export LD_LIBRARY_PATH="/nix/store/chqq8mpmpyfi9kgsngya71akv5xicn03-gcc-15.2.0-lib/lib:/tmp/nvidia_libs"
#   source /tmp/sd_venv/bin/activate
#   python docs/steam/gen_master.py

set -euo pipefail

MASTER="docs/steam/parts/master_3840x2560.png"
OUT="docs/steam/final"
FONT="$HOME/.fonts/AUGUSTUS.TTF"

mkdir -p "$OUT"

# 各アセットサイズに応じたピクセルアート化を行う
# LANCZOS で縮小 → Point (NEAREST) で拡大し、ピクセルブロック感を出す
# 各アセットサイズに応じたピクセルアート化を ImageMagick で行う
# LANCZOS で縮小 → Point (NEAREST) で拡大し、ピクセルブロック感を出す
pixelate() {
  local input=$1 w=$2 h=$3 scale=$4 output=$5
  local sw=$((w / scale)) sh=$((h / scale))
  magick "$input" \
    -filter Lanczos -resize "${sw}x${sh}!" \
    -filter Point -resize "${w}x${h}!" \
    "$output"
}

# --- 大きい画像: ダンジョン背景 ---

# Library Hero 3840x1240 (横長パノラマ、中央から切り出し)
magick "$MASTER" -gravity center -crop 3840x1240+0+0 +repage /tmp/steam_crop.png
pixelate /tmp/steam_crop.png 3840 1240 4 /tmp/steam_crop.png
magick /tmp/steam_crop.png -modulate 60 "$OUT/library_hero.png"
echo "library_hero.png (3840x1240)"

# Page Background 1438x810 (中央クロップ → リサイズ)
# 比率 1438:810 = 1.776:1 → 3840x2162 からクロップ
magick "$MASTER" -gravity center -crop 3840x2162+0+0 +repage -resize 1438x810! /tmp/steam_crop.png
pixelate /tmp/steam_crop.png 1438 810 3 /tmp/steam_crop.png
magick /tmp/steam_crop.png -modulate 60 "$OUT/page_background.png"
echo "page_background.png (1438x810)"

# --- ロゴ付きカプセル: ダンジョン背景をクロップ・暗化 + ロゴ ---

generate_capsule() {
  local w=$1 h=$2 ps=$3 fname=$4
  local crop_w=$5 crop_h=$6 scale=$7 logo_gravity=$8

  magick "$MASTER" -gravity center -crop "${crop_w}x${crop_h}+0+0" +repage \
    -resize "${w}x${h}!" /tmp/steam_crop.png
  pixelate /tmp/steam_crop.png "$w" "$h" "$scale" /tmp/steam_crop.png
  # north の場合、高さの 1/5 をマージンとする
  local y_off=0
  if [ "$logo_gravity" = "north" ]; then
    y_off=$((h / 10))
  fi

  magick /tmp/steam_crop.png \
    -modulate 40 \
    -gravity "$logo_gravity" \
    -font "$FONT" -pointsize "$ps" \
    -fill 'rgba(0,0,0,0.6)' -annotate "+2+$((y_off + 2))" "Ruins" \
    -fill '#c8d8f0' -annotate "+0+${y_off}" "Ruins" \
    "$OUT/$fname"

  echo "$fname (${w}x${h})"
}

# 各カプセルのクロップサイズはマスター 3840x2560 からアスペクト比に合わせて計算
#                                       w    h    ps  filename              crop_w crop_h scale gravity
generate_capsule                        462  174  110  small_capsule.png     3840   1446   2    center
generate_capsule                        920  430  180  header_capsule.png    3840   1794   3    center
generate_capsule                        920  430  180  library_header.png    3840   1794   3    center
generate_capsule                       1232  706  280  main_capsule.png      3840   2200   3    center
generate_capsule                        748  896  200  vertical_capsule.png  2138   2560   3    north
generate_capsule                        600  900  170  library_capsule.png   1707   2560   3    north

# --- Library Logo: 透明背景 + ロゴ ---

magick -size 1280x720 xc:none \
  -gravity center \
  -font "$FONT" -pointsize 300 \
  -stroke black -strokewidth 6 \
  -fill '#c8d8f0' -annotate +0+0 "Ruins" \
  -stroke none \
  -fill '#c8d8f0' -annotate +0+0 "Ruins" \
  "$OUT/library_logo.png"
echo "library_logo.png (1280x720)"

echo ""
echo "All assets generated in $OUT/"
