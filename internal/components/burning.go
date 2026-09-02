package components

import (
	"github.com/kijimaD/ruins/internal/consts"
)

// Burning は今まさに燃えているエンティティに付く状態。地面のタイルに宿る。
// 燃料を貯め込まず残量だけを持ち、くべた燃料は Remaining へ畳み込む。
// Burning がある間だけ同じエンティティの HeatSource が暖房として効く。
type Burning struct {
	Remaining consts.Turn // 残りの燃焼ターン数。毎ターン減り、0 以下で Burning が外れて火が消える。着火や給油でここへ燃料が畳み込まれる
}
