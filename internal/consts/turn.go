package consts

import (
	"fmt"
)

// Turn はターンで数える量。1 ＝ 1ターン。
//
// タイルの Tile やチャンクの Chunk と同じく単位を型で区別するスカラー。総経過ターン数・ターン番号・
// 経過ターン数など「ターンで数える量」を裸の int と取り違えないようにする。
// タイルやチャンクとの誤った演算を型が弾く。軸のような向きは持たない。
type Turn int

// String は時計アイコンを先頭に付けて「N ターン」を表す。燃焼の残ターンなど時間量の表示に使う。
// 通貨や熱量と同じく記号を先頭にし半角スペースで区切る
func (t Turn) String() string {
	return fmt.Sprintf("%s %d", IconClock, int(t))
}

// StringDelta は増分を符号付きで表す。給油で「+N ターン延びる」といった差分の表示に使う
func (t Turn) StringDelta() string {
	return fmt.Sprintf("%s +%d", IconClock, int(t))
}
