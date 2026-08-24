#!/bin/bash
# Coldward のロゴを決定的に生成する。
#
# 構造:
#   - 頭文字 C を大きく
#   - 残りの OLDWARD をその半分の高さで上寄せに1行
#   - 下にライン + 小さな副題
# 各要素をトリムして実寸から座標を計算するので、点サイズ・色・字間を変えても
# 同じ手順で再現的に組み直せる。手で1pxずつ置く工程を持たない。
#
# 依存:
#   - ImageMagick (magick)
#   - このディレクトリの fonts に vendored したフォント。Iceland (頭文字と本体) と
#     Staatliches (副題)。いずれも SIL Open Font License。各ディレクトリの OFL.txt 参照
#
# 生成スクリプトと成果物は同じディレクトリに置く。最終的に compose がこれら各パーツを組み合わせる。
#
# 使い方: bash docs/steam/logo/gen_logo.sh
# 出力 (このスクリプトと同じディレクトリ):
#   logo.png       透過・高解像のマスタ
#   logo_pixel.png 背景のドット絵と画素グリッドを合わせたピクセル版

set -euo pipefail
cd "$(dirname "$0")"

FONTS=fonts
MAIN_FONT=$FONTS/Iceland/Iceland-Regular.ttf   # 頭文字と本体。角張った低曲率のレトロ書体
STAAT=$FONTS/Staatliches/Staatliches-Regular.ttf
OUT=.
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# --- パラメータ。ここだけ変えれば見た目を振れる ---
INITIAL="C"                          # 大きな頭文字
REST="OLDWARD"                       # 残りの文字
SUBTITLE="NO WARMTH, NO RETURN"      # 副題
PT_BIG=320                           # 頭文字の点サイズ
PT_REST=160                          # 残りの点サイズ。頭文字の半分
KERN_REST=4                          # 残りの字間
FILL_TOP='#f6fbff'                   # 氷グラデの上。雪の白
FILL_BOT='#8aabce'                   # 氷グラデの下。鋼の青
OUTLINE='#0d1a2e'                    # 濃紺の縁
SHADOW='#03060d'                     # 影
LINE_COLOR='#a6e2f2'                 # ライン。氷シアン
SUB_COLOR='#c7d8e8'                  # 副題。淡い寒色
PIXEL_BLOCK=5                        # ピクセル1ブロックのpx。背景に合わせる
PIXEL_COLORS=44                      # ピクセル版のパレット数
GAP=6                                # 頭文字と残りの間隔。詰めて右の空きを消す
LINE_THICK=24                        # 下線の太さ。左端の高さ
LINE_GAP=4                           # OLDWARD 下端と下線の間隔。小さいほど近づく
SUB_PT=30                            # 副題の点サイズ。小さめに抑える

# 氷処理した1語を作る。引数: text point kern outlineDiskRadius outPath
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

treat "$INITIAL" "$PT_BIG"  0           8 "$TMP/C.png"
treat "$REST"    "$PT_REST" "$KERN_REST" 4 "$TMP/OW.png"

Cw=$(identify -format %w "$TMP/C.png"); Ch=$(identify -format %h "$TMP/C.png")
Ow=$(identify -format %w "$TMP/OW.png"); Oh=$(identify -format %h "$TMP/OW.png")

# 下線は OLDWARD の幅ぶん引き、右へ尖らせて方向性を出す
LINE_LEN=$Ow

# 副題。淡い寒色のままだと明るい背景で埋もれるので、本体と同じく細い濃紺の縁と薄影を付ける。
# 小さい文字なので縁は細めの Disk:2 に留める。潰れを避ける。
magick -background none -fill "$SUB_COLOR" -font "$STAAT" -kerning 14 -pointsize "$SUB_PT" label:"$SUBTITLE" "$TMP/subtxt.png"
magick "$TMP/subtxt.png" -bordercolor none -border 8 "$TMP/subtxt.png"
subsz=$(identify -format '%wx%h' "$TMP/subtxt.png")
magick "$TMP/subtxt.png" -alpha extract "$TMP/suba.png"
magick -size "$subsz" xc:"$OUTLINE" \( "$TMP/suba.png" -morphology Dilate Disk:2 \) -compose CopyOpacity -composite "$TMP/subk.png"
magick -size "$subsz" xc:"$SHADOW" \( "$TMP/suba.png" -morphology Dilate Disk:1 -blur 0x2 \) -compose CopyOpacity -composite "$TMP/subs.png"
magick -size "$subsz" xc:none \
  \( "$TMP/subs.png" \) -gravity center -geometry +0+2 -compose Over -composite \
  \( "$TMP/subk.png" \) -compose Over -composite \
  \( "$TMP/subtxt.png" \) -compose Over -composite -trim +repage "$TMP/tag.png"
Tw=$(identify -format %w "$TMP/tag.png")
if [ "$Tw" -gt "$LINE_LEN" ]; then
  magick "$TMP/tag.png" -resize "${LINE_LEN}x" "$TMP/tag.png"
  Tw=$(identify -format %w "$TMP/tag.png")
fi
Th=$(identify -format %h "$TMP/tag.png")

# レイアウト計算。頭文字の右へ間隔GAPで残りを上寄せ、その下に下線、さらに下に副題
XOW=$((Cw + GAP))
LINE_Y=$((Oh + LINE_GAP))
# 副題の下端を頭文字 C の下端に完全に合わせる。C は影を持ち Ch に影ぶんが含まれるので、
# 影を除いた縁つき C の実高 Cbot を測り、そこへ副題の下端を揃える。Disk:8 は C の縁と一致させる
Cbot=$(magick -background none -fill white -font "$MAIN_FONT" -pointsize "$PT_BIG" label:"$INITIAL" -alpha extract -morphology Dilate Disk:8 -trim +repage -format %h info:)
TAG_Y=$((Cbot - Th))
CANVAS_W=$((XOW + LINE_LEN + 40))
CANVAS_H=$Ch

# 下線は矩形のまま、右へ向かって透明度を落として方向を出す。左が不透明、右端で透明
magick -size "${LINE_LEN}x${LINE_THICK}" xc:"$LINE_COLOR" \
  \( -size "${LINE_LEN}x${LINE_THICK}" xc: -sparse-color barycentric "0,0 white ${LINE_LEN},0 black" \) \
  -compose CopyOpacity -composite "$TMP/line.png"

magick -size "${CANVAS_W}x${CANVAS_H}" xc:none \
  \( "$TMP/C.png" \)  -gravity NorthWest -geometry +0+0 -compose Over -composite \
  \( "$TMP/OW.png" \) -gravity NorthWest -geometry +${XOW}+0 -compose Over -composite \
  \( "$TMP/line.png" \) -gravity NorthWest -geometry +${XOW}+${LINE_Y} -compose Over -composite \
  \( "$TMP/tag.png" \) -gravity NorthWest -geometry +${XOW}+${TAG_Y} -compose Over -composite \
  -trim +repage "$OUT/logo.png"

# ドット絵化。縮小 → パレット量子化 → 最近傍で拡大し、背景の画素グリッドに合わせる
FW=$(identify -format %w "$OUT/logo.png")
magick "$OUT/logo.png" -filter Box -resize $((FW / PIXEL_BLOCK))x -colors "$PIXEL_COLORS" -filter Point -resize "${FW}x" "$OUT/logo_pixel.png"

echo "logo.png       $(identify -format '%wx%h' "$OUT/logo.png")"
echo "logo_pixel.png $(identify -format '%wx%h' "$OUT/logo_pixel.png")"
echo "written to $OUT/"
