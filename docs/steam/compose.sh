#!/bin/bash
# Steam アセットの最終合成。各パーツを組み合わせて配布用アセットを作る。
# 背景マスタを各カプセルサイズへ切り出し、ロゴを重ねて generated/ へ出力する。
#
# パーツは各ディレクトリに生成スクリプトと成果物が同居する:
#   - background/ 背景マスタ (gen_master.py)
#   - logo/       ロゴ (gen_logo.sh)
# この compose がそれらを最終アセットへ束ねる。
#
# 使い方: bash docs/steam/compose.sh
#
# 依存: ImageMagick (magick)
#
# 注記: 背景マスタは旧 SD ダンジョンのまま。ドット絵へ刷新する際、この compose も新パーツに
# 合わせて調整する。ロゴは完成品 logo/logo.png を加工せず重ねる。

set -euo pipefail

MASTER="docs/steam/background/master_3840x2560.png"
MASTER_VERT="docs/steam/background/master_vert_2560x3840.png"
OUT="docs/steam/generated"
# logo/gen_logo.sh の完成品ロゴ。氷塗り・縁・影を内包するので加工せずそのまま重ねる
LOGO_PNG="docs/steam/logo/logo.png"

mkdir -p "$OUT"

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
# 完成品ロゴを背景に重ねる。ロゴは縁・影・氷塗りを内包するので加工しない。
# 幅 max_w と 高さ logo_h の箱にアスペクト維持で収めてから合成する
render_logo() {
  local w=$1 h=$2 logo_h=$3 gravity=$4 y_off=$5 output=$6

  local max_w=$(( w * 82 / 100 ))
  magick "$LOGO_PNG" -resize "${max_w}x${logo_h}" /tmp/steam_logo.png

  magick "$output" \
    -gravity "$gravity" \
    /tmp/steam_logo.png -geometry "+0+${y_off}" -compose Over -composite \
    "$output"
}

# --- ロゴ付きカプセル: ダンジョン背景をクロップ・暗化 + ロゴ ---

generate_capsule() {
  local w=$1 h=$2 fname=$3
  local crop_w=$4 crop_h=$5 scale=$6 logo_gravity=$7
  local master_src=${8:-$MASTER}

  magick "$master_src" -gravity center -crop "${crop_w}x${crop_h}+0+0" +repage \
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
  # north の場合、高さの 1/10 をマージンとする
  local y_off=0
  if [ "$logo_gravity" = "north" ]; then
    y_off=$((h / 10))
  fi

  render_logo "$w" "$h" "$logo_h" "$logo_gravity" "$y_off" "$OUT/$fname"

  echo "$fname (${w}x${h})"
}

# 各カプセルのクロップサイズはマスターからアスペクト比に合わせて計算
# 横長はマスター (3840x2560)、縦長は縦長マスター (2560x3840) を使用する
#                                       w    h    filename              crop_w crop_h scale gravity  master
generate_capsule                        462  174  small_capsule.png     3840   1446   2    center
generate_capsule                        920  430  header_capsule.png    3840   1794   3    center
generate_capsule                        920  430  library_header.png    3840   1794   3    center
generate_capsule                       1232  706  main_capsule.png      3840   2200   3    center
generate_capsule                        748  896  vertical_capsule.png  2560   3066   2    north   "$MASTER_VERT"
generate_capsule                        600  900  library_capsule.png   2560   3840   2    north   "$MASTER_VERT"

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
