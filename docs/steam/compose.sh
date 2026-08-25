#!/bin/bash
# Steam アセットの最終合成。背景マスタを各カプセルサイズへ切り出し、ロゴを重ねて generated/ へ出す。
# パーツは各ディレクトリに同居する: background/(gen_master.py) と logo/(gen_logo.sh)。
# 使い方: nix-shell docs/steam/shell.nix 内で bash docs/steam/compose.sh  依存: ImageMagick

set -euo pipefail

# マスターは横1枚だけ。縦カプセルもこの1枚をポートレートに切り出す。
MASTER="docs/steam/background/master_3840x2560.png"
OUT="docs/steam/generated"
# 完成品ロゴ。縁・影を内包するので加工せずそのまま重ねる
LOGO_PNG="docs/steam/logo/logo.png"

mkdir -p "$OUT"

# --- 大きい画像 ---

# Library Hero 3840x1240 横長パノラマ、中央から切り出し
magick "$MASTER" -gravity center -crop 3840x1240+0+0 +repage "$OUT/library_hero.png"
echo "library_hero.png (3840x1240)"

# Page Background 1438x810 中央クロップ → リサイズ
# 比率 1438:810 = 1.776:1 → 3840x2162 からクロップ
magick "$MASTER" -gravity center -crop 3840x2162+0+0 +repage -resize 1438x810! "$OUT/page_background.png"
echo "page_background.png (1438x810)"

# --- ロゴ描画関数 ---
# 完成品ロゴを背景に重ねる。ロゴは縁・影・氷塗りを内包するので加工しない。
# 幅 max_w と 高さ logo_h の箱にアスペクト維持で収めてから合成する
render_logo() {
  local w=$1 h=$2 logo_h=$3 gravity=$4 y_off=$5 output=$6

  local max_w=$(( w * 82 / 100 ))
  magick "$LOGO_PNG" -resize "${max_w}x${logo_h}" /tmp/steam_logo.png

  # 宛先を sRGB へ昇格してから合成する。透明ベース xc:none が Gray になり色が落ちるのを防ぐ
  magick "$output" -colorspace sRGB \
    -gravity "$gravity" \
    /tmp/steam_logo.png -geometry "+0+${y_off}" -compose Over -composite \
    "$output"
}

# --- ロゴ付きカプセル: マスタをクロップ・リサイズ + ロゴ ---

generate_capsule() {
  local w=$1 h=$2 fname=$3
  local crop_w=$4 crop_h=$5 logo_gravity=$6
  # クロップ基準と横オフセット。縦カプセルはキューブが左下にあるため左寄りに切り出す
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
# すべて横マスター1枚から中央で切り出す。キューブがマスター中央にあるので、縦長も中央クロップで
# キューブが真ん中に来る。
#                                       w    h    filename              crop_w crop_h logo_grav
generate_capsule                        462  174  small_capsule.png     3840   1446   center
generate_capsule                        920  430  header_capsule.png    3840   1794   center
generate_capsule                        920  430  library_header.png    3840   1794   center
generate_capsule                       1232  706  main_capsule.png      3840   2200   center
generate_capsule                        748  896  vertical_capsule.png  2137   2560   north
generate_capsule                        600  900  library_capsule.png   1707   2560   north

# --- ゲームタイトル用画像 960x720 ---
# シネマ配置。ロゴを左上、メニューは左下(ゲーム側で描画)に置き、主役のキューブを右に残す。
# 背景はストアと同じキーアートを、キューブが右へ来るようズームして切り出す。
magick "$MASTER" -crop 2400x1800+240+680 +repage -resize 960x720! /tmp/title_bg.png
# 下部と左にダークグラデのスクリムを敷く。左下のメニュー域を暗くして可読性を確保する。
magick /tmp/title_bg.png \
  \( -size 960x720 gradient:none-'#060a14' \) -compose over -composite \
  \( -size 720x960 gradient:'#060a1466'-none -rotate 90 \) -compose over -composite \
  "assets/file/textures/bg/title1_.png"
# ロゴを左上へ。幅は画面の約55%。メニューは main_menu.go が左下へ左寄せで描く。
magick "$LOGO_PNG" -resize 500x /tmp/title_logo.png
magick "assets/file/textures/bg/title1_.png" /tmp/title_logo.png -gravity NorthWest -geometry +48+56 -compose over -composite "assets/file/textures/bg/title1_.png"
echo "title1_.png (960x720)"

# --- Library Logo: 透明背景 + ロゴ ---

magick -size 1280x720 xc:none "$OUT/library_logo.png"
render_logo 1280 720 180 center 0 "$OUT/library_logo.png"
echo "library_logo.png (1280x720)"

echo ""
echo "All assets generated in $OUT/"
