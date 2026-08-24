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
#   logo.png 透過・高解像のマスタ

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
GAP=6                                # 頭文字と残りの間隔。詰めて右の空きを消す
LINE_THICK=24                        # 帯の太さ。左端の高さ
LINE_GAP=4                           # OLDWARD 下端と帯の間隔。小さいほど近づく
LINE_SLANT=22                        # 帯の両端の斜めカット量。上辺を右へずらす横せん断
SUB_PT=30                            # 副題の点サイズ。小さめに抑える
SUB_GAP=10                           # 帯の下端と副題の間隔。重なりを避ける

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

# レイアウト計算。頭文字の右へ間隔GAPで残りを上寄せ、その下に帯、さらに下に副題
XOW=$((Cw + GAP))
LINE_Y=$((Oh + LINE_GAP))
# 副題は帯の下端より下へ置き、帯と重ならないようにする。帯の最下点は左端の LINE_Y+LINE_THICK。
TAG_Y=$((LINE_Y + LINE_THICK + SUB_GAP))
CANVAS_W=$((XOW + LINE_LEN + 40))
# 副題が C の下端を超えるならキャンバスを下へ伸ばす。超えなければ C の高さのまま
CANVAS_H=$Ch
if [ $((TAG_Y + Th + 4)) -gt "$CANVAS_H" ]; then
  CANVAS_H=$((TAG_Y + Th + 4))
fi

# 帯は水平のまま両端を斜めにカットした平行四辺形。上下辺は水平、左右辺が「/」に斜め。
# 上辺を下辺より右へ LINE_SLANT ずらす横せん断で作る。スピードストライプ状の動きを出す。
# さらに右へ向かって透明度を落とし方向を出す。左が不透明、右端で透明。
# 透明化はポリゴンのアルファに横グラデを乗算して行う。CopyOpacity で全面置換すると外側の三角も出るため。
LW=$((LINE_LEN + LINE_SLANT))
magick -size "${LW}x${LINE_THICK}" xc:none -fill "$LINE_COLOR" \
  -draw "polygon 0,${LINE_THICK} ${LINE_SLANT},0 ${LW},0 ${LINE_LEN},${LINE_THICK}" "$TMP/pg.png"
magick "$TMP/pg.png" -alpha extract "$TMP/pga.png"
magick -size "${LW}x${LINE_THICK}" xc: -sparse-color barycentric "0,0 white ${LW},0 black" "$TMP/grad.png"
magick "$TMP/pga.png" "$TMP/grad.png" -compose Multiply -composite "$TMP/newa.png"
magick "$TMP/pg.png" "$TMP/newa.png" -compose CopyOpacity -composite "$TMP/line.png"

magick -size "${CANVAS_W}x${CANVAS_H}" xc:none \
  \( "$TMP/C.png" \)  -gravity NorthWest -geometry +0+0 -compose Over -composite \
  \( "$TMP/OW.png" \) -gravity NorthWest -geometry +${XOW}+0 -compose Over -composite \
  \( "$TMP/line.png" \) -gravity NorthWest -geometry +${XOW}+${LINE_Y} -compose Over -composite \
  \( "$TMP/tag.png" \) -gravity NorthWest -geometry +${XOW}+${TAG_Y} -compose Over -composite \
  -trim +repage "$OUT/logo.png"

echo "logo.png $(identify -format '%wx%h' "$OUT/logo.png")"
echo "written to $OUT/"
