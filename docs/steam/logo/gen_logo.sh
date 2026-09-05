#!/bin/bash
# Coldward のロゴを決定的に生成する。頭文字 C を大きく、OLDWARD を上、副題を C の下端へ揃え、氷の帯を両者の中間へ。
# フラットな字面に CRT の色収差を重ね、寒色でまとめる。縁・影・グラデは使わない。
# 各要素をトリムして実寸から座標を計算するので、点サイズ・色・字間を変えても再現的に組み直せる。
# 依存: ImageMagick と fonts の Iceland/Staatliches。いずれも SIL OFL、各 OFL.txt 参照。
# 使い方: bash docs/steam/logo/gen_logo.sh  出力: logo.png 透過・高解像
# グローはロゴに焼かず、表示側の CSS フィルタに任せてレイアウトを安定させる。

set -euo pipefail
cd "$(dirname "$0")"

FONTS=fonts
MAIN_FONT=$FONTS/Iceland/Iceland-Regular.ttf # 頭文字と本体。角張った低曲率のレトロ書体
STAAT=$FONTS/Staatliches/Staatliches-Regular.ttf
OUT=.
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# --- パラメータ。ここだけ変えれば見た目を振れる ---
INITIAL="C"                     # 大きな頭文字
REST="OLDWARD"                  # 残りの文字
SUBTITLE="NO WARMTH, NO RETURN" # 副題
PT_BIG=320                      # 頭文字の点サイズ
PT_REST=160                     # 残りの点サイズ。頭文字の半分
KERN_REST=4                     # 残りの字間
FILL='#e6f0f7'                  # フラットなフロスト白。塗りは単色
ABERR_CYAN='#4fd2ff'            # 色収差のシアン。右へずらす
ABERR_RED='#ff4f70'             # 色収差のレッド。左へずらす
ABERR=4                         # RGB のずらし量 px。CRT の色ずれ
LINE_COLOR='#a6e2f2'            # ライン。氷シアン
SUB_COLOR='#bcd4e6'             # 副題。淡い寒色
GAP=6                           # 頭文字と残りの間隔。詰めて右の空きを消す
LINE_THICK=24                   # 帯の太さ。左端の高さ
LINE_SLANT=22                   # 帯の両端の斜めカット量。上辺を右へずらす横せん断
LINE_SUB_GAP=8                  # 帯の下端と副題上端の間隔。小さいほど帯が副題へ近づく
SUB_PT=30                       # 副題の点サイズ。小さめに抑える

# フラットな字を CRT の色収差付きで作る。引数: text point kern outPath
# シアンを右、レッドを左へ数px ずらし、中央にフロスト白を重ねる。縁の外だけ色が覗く
treat() {
	local t="$1" p="$2" k="$3" out="$4" sz
	magick -background none -fill white -font "$MAIN_FONT" -kerning "$k" -pointsize "$p" label:"$t" "$TMP/c.png"
	magick "$TMP/c.png" -bordercolor none -border 20 "$TMP/c.png"
	sz=$(identify -format '%wx%h' "$TMP/c.png")
	magick "$TMP/c.png" -alpha extract "$TMP/a.png"
	# 単色でマスクを塗る。フラット塗りと色収差用のシアン・レッド
	magick -size "$sz" xc:"$FILL" \( "$TMP/a.png" \) -compose CopyOpacity -composite "$TMP/f.png"
	magick -size "$sz" xc:"$ABERR_CYAN" \( "$TMP/a.png" \) -compose CopyOpacity -composite "$TMP/cy.png"
	magick -size "$sz" xc:"$ABERR_RED" \( "$TMP/a.png" \) -compose CopyOpacity -composite "$TMP/rd.png"
	# シアン(右) → レッド(左) → フロスト白(中央) の順に重ねる
	magick -size "$sz" xc:none \
		\( "$TMP/cy.png" \) -gravity center -geometry +${ABERR}+0 -compose Over -composite \
		\( "$TMP/rd.png" \) -gravity center -geometry -${ABERR}+0 -compose Over -composite \
		\( "$TMP/f.png" \) -gravity center -geometry +0+0 -compose Over -composite \
		-trim +repage "$out"
}

treat "$INITIAL" "$PT_BIG" 0 "$TMP/C.png"
treat "$REST" "$PT_REST" "$KERN_REST" "$TMP/OW.png"

# 幅と高さは1回の identify でまとめて取る。2回呼ぶと画像の再読み込みが二重になる
read -r Cw Ch <<<"$(identify -format '%w %h' "$TMP/C.png")"
read -r Ow Oh <<<"$(identify -format '%w %h' "$TMP/OW.png")"

# 下線は OLDWARD の幅ぶん引き、右へ尖らせて方向性を出す
LINE_LEN=$Ow

# 副題。フラットな淡い寒色。縁も影も付けない
magick -background none -fill "$SUB_COLOR" -font "$STAAT" -kerning 14 -pointsize "$SUB_PT" label:"$SUBTITLE" "$TMP/tag.png"
magick "$TMP/tag.png" -trim +repage "$TMP/tag.png"
Tw=$(identify -format %w "$TMP/tag.png")
if [ "$Tw" -gt "$LINE_LEN" ]; then
	magick "$TMP/tag.png" -resize "${LINE_LEN}x" "$TMP/tag.png"
fi
# リサイズ確定後に幅高さをまとめて取る
read -r Tw Th <<<"$(identify -format '%w %h' "$TMP/tag.png")"

# レイアウト計算。C の右へ間隔GAPで OLDWARD を上寄せ。副題の下端を C の下端に揃え、
# 帯は副題の上端から LINE_SUB_GAP だけ空けた直上へ置く。全要素が C の高さに収まる
XOW=$((Cw + GAP))
TAG_Y=$((Ch - Th))
LINE_Y=$((TAG_Y - LINE_SUB_GAP - LINE_THICK))
CANVAS_W=$((XOW + LINE_LEN + 40))
CANVAS_H=$Ch

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
	\( "$TMP/C.png" \) -gravity NorthWest -geometry +0+0 -compose Over -composite \
	\( "$TMP/OW.png" \) -gravity NorthWest -geometry +${XOW}+0 -compose Over -composite \
	\( "$TMP/line.png" \) -gravity NorthWest -geometry +${XOW}+${LINE_Y} -compose Over -composite \
	\( "$TMP/tag.png" \) -gravity NorthWest -geometry +${XOW}+${TAG_Y} -compose Over -composite \
	-trim +repage "$OUT/logo.png"

echo "logo.png $(identify -format '%wx%h' "$OUT/logo.png")"
echo "written to $OUT/"
