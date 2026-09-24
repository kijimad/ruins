package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// dangerDaysPerLevel は危険度が1段上がる経過日数。値は暫定で、実プレイの伸びを測って振り直す。
const dangerDaysPerLevel = 3

// DangerLevelForDay は経過日数から危険度を返す純関数。危険度は1始まりで最小は1。
// 単調非減少で、同じ入力は常に同じ危険度を返す。バランス導出が world 抜きで時系列の
// 難易度カーブを引けるよう公開する。
func DangerLevelForDay(days int) consts.Danger {
	if days < 0 {
		days = 0
	}
	return 1 + consts.Danger(days)/dangerDaysPerLevel
}

// DangerLevelAt は world から経過日数を引いて危険度を返す。
func DangerLevelAt(world w.World) consts.Danger {
	return DangerLevelForDay(GetGameTime(world).GetDayNumber())
}

// dangerChunksPerLevel は危険度が1段上がる北進チャンク数。値は暫定で、実プレイの伸びを測って振り直す。
const dangerChunksPerLevel = 3

// DangerLevelForDepth は北へ chunksNorth 進んだ場所の危険度を返す純関数。空間の難易度勾配。
// world を引かず座標で決めるので生成の再訪一致を壊さない。日数版 DangerLevelForDay と対をなす。
func DangerLevelForDepth(chunksNorth consts.Chunk) consts.Danger {
	if chunksNorth < 0 {
		chunksNorth = 0
	}
	return 1 + consts.Danger(chunksNorth)/dangerChunksPerLevel
}
