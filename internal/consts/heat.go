package consts

import (
	"fmt"
)

// Heat は燃料の熱量。整数で正確に扱い、表示整形は String に閉じる。
// 燃焼ターン数や重量と混同しないよう、素の int と型で区別する。
// 効率で割り引くと実際の燃焼ターン数になる。常に非負で、1 は効率100%で1ターンぶんの燃焼に相当する。
type Heat int

// String は炎アイコンを先頭に付けて整形する。通貨や重量と同じく記号を先頭にし半角スペースで区切る。
func (h Heat) String() string {
	return fmt.Sprintf("%s %d", IconFire, int(h))
}

// BurnTurns は効率パーセントで割り引いた燃焼ターン数を返す。
// 熱量から実際に燃える時間への変換をここへ集約し、Turn を返して熱量との境界を型で明示する。
func (h Heat) BurnTurns(efficiencyPct int) Turn {
	return Turn(int(h) * efficiencyPct / 100)
}
