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
LOGO_SVG="docs/steam/parts/logo.svg"
FONT="$HOME/.fonts/AUGUSTUS.TTF"

mkdir -p "$OUT"

# SVG ロゴを一度だけ大きな透過 PNG にレンダリングする
LOGO_PNG="docs/steam/parts/logo.png"
magick -background none "$LOGO_SVG" \
  -fuzz 100% -fill white -opaque black \
  "$LOGO_PNG"

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

# --- ロゴ描画関数 ---
# 光彩 + アウトライン + グラデーション塗りでロゴを生成する
# 引数: 幅 高さ pointsize gravity y_offset 出力ファイル
render_logo() {
  local w=$1 h=$2 logo_h=$3 gravity=$4 y_off=$5 output=$6

  # マスター PNG からリサイズ
  local logo_w=$(( logo_h * 3600 / 1050 ))

  magick "$LOGO_PNG" -resize "${logo_w}x${logo_h}" /tmp/steam_logo.png

  # 影レイヤーを作成
  magick /tmp/steam_logo.png \
    -fuzz 100% -fill 'rgba(0,0,0,0.5)' -opaque white \
    /tmp/steam_shadow.png

  # 背景に合成: 影 → ロゴ
  magick "$output" \
    -gravity "$gravity" \
    /tmp/steam_shadow.png -geometry "+3+$((y_off + 3))" -compose Over -composite \
    /tmp/steam_logo.png -geometry "+0+${y_off}" -compose Over -composite \
    "$output"
}

# --- ロゴ付きカプセル: ダンジョン背景をクロップ・暗化 + ロゴ ---

generate_capsule() {
  local w=$1 h=$2 fname=$3
  local crop_w=$4 crop_h=$5 scale=$6 logo_gravity=$7

  magick "$MASTER" -gravity center -crop "${crop_w}x${crop_h}+0+0" +repage \
    -resize "${w}x${h}!" /tmp/steam_crop.png
  pixelate /tmp/steam_crop.png "$w" "$h" "$scale" /tmp/steam_crop.png
  # 暗すぎると端が黒枠に見えて Steam にリジェクトされるため、控えめに暗化する
  magick /tmp/steam_crop.png -modulate 60 "$OUT/$fname"

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
  local max_h=$(( w * 80 / 100 * 1050 / 3600 ))
  if [ "$logo_h" -gt "$max_h" ]; then
    logo_h=$max_h
  fi
  # north の場合、高さの 1/10 をマージンとする
  local y_off=0
  if [ "$logo_gravity" = "north" ]; then
    y_off=$((h / 10))
  fi

  render_logo "$w" "$h" "$logo_h" "$logo_gravity" "$y_off" "$OUT/$fname"

  echo "$fname (${w}x${h})"
}

# 各カプセルのクロップサイズはマスター 3840x2560 からアスペクト比に合わせて計算
#                                       w    h    filename              crop_w crop_h scale gravity
generate_capsule                        462  174  small_capsule.png     3840   1446   2    center
generate_capsule                        920  430  header_capsule.png    3840   1794   3    center
generate_capsule                        920  430  library_header.png    3840   1794   3    center
generate_capsule                       1232  706  main_capsule.png      3840   2200   3    center
generate_capsule                        748  896  vertical_capsule.png  2138   2560   3    north
generate_capsule                        600  900  library_capsule.png   1707   2560   3    north

# --- ゲームタイトル用画像 960x720 ---
# 比率 960:720 = 4:3 → 3840x2880 だがマスターは 2560 高なので 3413x2560 からクロップ
magick "$MASTER" -gravity center -crop 3413x2560+0+0 +repage -resize 960x720! /tmp/steam_crop.png
pixelate /tmp/steam_crop.png 960 720 3 /tmp/steam_crop.png
magick /tmp/steam_crop.png -modulate 50 "assets/file/textures/bg/title1_.png"
render_logo 960 720 180 north 72 "assets/file/textures/bg/title1_.png"
echo "title1_.png (960x720)"

# --- Library Logo: 透明背景 + ロゴ ---

magick -size 1280x720 xc:none "$OUT/library_logo.png"
render_logo 1280 720 180 center 0 "$OUT/library_logo.png"
echo "library_logo.png (1280x720)"

echo ""
echo "All assets generated in $OUT/"
