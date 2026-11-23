#!/bin/bash
# Coldward のロゴを決定的に生成する。縦長の極太書体で COLDWARD を広い字間に組み、下へ細い罫で挟んだ小さな副題を置く。
# 塗りは単色で、縁取り・グラデ・色収差は使わない。背景から浮かせるのは、広く薄い暗い暈しだけに任せる。
# 依存: ImageMagick と fonts の Bebas Neue と Jost。いずれも SIL OFL、各 OFL.txt 参照。
# 使い方: bash docs/steam/logo/gen_logo.sh  出力: logo.png 透過・高解像

set -euo pipefail
cd "$(dirname "$0")"

TITLE_FONT=fonts/BebasNeue/BebasNeue-Regular.ttf # 縦長の極太。字間を空けても塊に見える
SUB_FONT=fonts/Jost/Jost-Regular.ttf             # 副題。幾何学的なサンセリフで題字の硬さに合わせる
OUT=.
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

# --- パラメータ。ここだけ変えれば見た目を振れる ---
TITLE="COLDWARD"
SUBTITLE="NO WARMTH, NO RETURN"
PT=220             # 題字の点サイズ
KERN=50            # 題字の字間
FILL='#eef3f6'     # 題字の塗り。わずかに寒色へ寄せた白
SUB_PT=$((PT / 8)) # 副題の点サイズ
SUB_KERN=$((SUB_PT / 2))
SUB_COLOR='#d3dde4' # 副題と罫の色。題字より一段落とす
RULE_THICK=2        # 罫の太さ
SUB_GAP=$((PT / 5)) # 題字と副題の間隔
HALO='55x22+0+4'    # 暗い暈しの濃さと広がり。硬い影でなく広く薄く敷く
HALO_COLOR='#03070d'
# 透明な下地。xc:none は Gray として作られ、重ねた色が落ちるので、青をわずかに持つ透明で RGB にする
CLEAR='rgba(0,0,1,0)'

magick -background "$CLEAR" -fill "$FILL" -font "$TITLE_FONT" -kerning "$KERN" -pointsize "$PT" label:"$TITLE" \
	-trim +repage "PNG32:$TMP/title.png"
Tw=$(identify -format '%w' "$TMP/title.png")

# 副題の両脇へ罫を引き、全体の幅を題字にそろえる
magick -background "$CLEAR" -fill "$SUB_COLOR" -font "$SUB_FONT" -kerning "$SUB_KERN" -pointsize "$SUB_PT" label:"$SUBTITLE" \
	-trim +repage "PNG32:$TMP/sub.png"
read -r Sw Sh <<<"$(identify -format '%w %h' "$TMP/sub.png")"
RULE_LEN=$(((Tw - Sw) / 2 - SUB_PT))
magick -size "${RULE_LEN}x${RULE_THICK}" xc:"$SUB_COLOR" "PNG32:$TMP/rule.png"
magick -size "${Tw}x${Sh}" xc:"$CLEAR" \
	\( "$TMP/rule.png" \) -gravity West -composite \
	\( "$TMP/sub.png" \) -gravity Center -composite \
	\( "$TMP/rule.png" \) -gravity East -composite \
	"PNG32:$TMP/subline.png"

magick "$TMP/title.png" "$TMP/subline.png" -background "$CLEAR" -gravity center -smush "$SUB_GAP" "PNG32:$TMP/base.png"

# 暈しが縁で切れないよう余白を足してから敷く
magick "$TMP/base.png" -bordercolor "$CLEAR" -border 60 \
	\( +clone -background "$HALO_COLOR" -shadow "$HALO" \) \
	+swap -background none -layers merge +repage "$OUT/logo.png"

echo "logo.png $(identify -format '%wx%h' "$OUT/logo.png")"
echo "written to $OUT/"
