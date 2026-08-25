#!/bin/bash
# Steam の Client Icon を決定的に生成する。ライブラリ等でタイトル横に出る小アイコン。
# ロゴと同じ設定で頭文字 C を描き、正方形の寒色暗背景に置く。単一の文字は 16px でも読める。
# 依存: ImageMagick と logo/fonts の Iceland 書体
# 出力: icon.png(512, Steam 用) / icon_256.png / icon.ico(16〜256 マルチサイズ, Windows 用)

set -euo pipefail
cd "$(dirname "$0")"

# --- gen_logo.sh と同一の設定 ---
MAIN_FONT=../logo/fonts/Iceland/Iceland-Regular.ttf  # 頭文字。角張った低曲率のレトロ書体
FILL_TOP='#e3e5ea'   # 単色オフホワイト。ほんの少し灰味。ロゴと同色
FILL_BOT='#e3e5ea'   # 単色オフホワイト。ほんの少し灰味。ロゴと同色
OUTLINE='#0d1a2e'    # 濃紺の縁
SHADOW='#03060d'     # 影
PT=420               # 頭文字の点サイズ。高解像で描いてから縮める

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# 氷処理した1文字を作る。gen_logo.sh の treat と同一。引数: text point kern outlineDiskRadius outPath
treat() {
  local t="$1" p="$2" k="$3" od="$4" out="$5" sz
  magick -background none -fill white -font "$MAIN_FONT" -kerning "$k" -pointsize "$p" label:"$t" "$TMP/c.png"
  magick "$TMP/c.png" -bordercolor none -border 30 "$TMP/c.png"
  sz=$(identify -format '%wx%h' "$TMP/c.png")
  magick "$TMP/c.png" -alpha extract "$TMP/a.png"
  # 濃紺の縁
  magick -size "$sz" xc:"$OUTLINE" \( "$TMP/a.png" -morphology Dilate Disk:"$od" \) -compose CopyOpacity -composite "$TMP/k.png"
  # 締まった影
  magick -size "$sz" xc:"$SHADOW" \( "$TMP/a.png" -morphology Dilate Disk:$((od / 2 + 1)) -blur 0x5 \) -compose CopyOpacity -composite "$TMP/s.png"
  # 氷グラデの塗り
  magick -size "$sz" gradient:"$FILL_TOP-$FILL_BOT" "$TMP/g.png"
  magick "$TMP/g.png" \( "$TMP/a.png" \) -compose CopyOpacity -composite "$TMP/f.png"
  # 影 → 縁 → 塗り の順に重ねる
  magick -size "$sz" xc:none \
    \( "$TMP/s.png" \) -gravity center -geometry +0+6 -compose Over -composite \
    \( "$TMP/k.png" \) -compose Over -composite \
    \( "$TMP/f.png" \) -compose Over -composite -trim +repage "$out"
}

treat "C" "$PT" 0 12 "$TMP/C.png"

# 正方形の寒色暗背景に C を中央配置する。明るい C が浮く。C は正方形の約8割に収める。
magick -size 512x512 radial-gradient:'#141d3e'-'#060810' "$TMP/bg.png"
magick "$TMP/bg.png" \
  \( "$TMP/C.png" -resize 400x400 \) -gravity center -compose Over -composite \
  icon.png

magick icon.png -resize 256x256 icon_256.png
magick icon.png -define icon:auto-resize=256,48,32,24,16 icon.ico

echo "icon.png     $(identify -format '%wx%h' icon.png)"
echo "icon_256.png $(identify -format '%wx%h' icon_256.png)"
echo "icon.ico     $(identify -format '%wx%h' 'icon.ico[0]') (multi-size)"
