package query

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// dangerDaysPerLevel は危険度が1段上がる経過日数。値は暫定で、実プレイの伸びを測って振り直す。
const dangerDaysPerLevel = 3

// dangerLevel は経過日数から危険度を返す純関数。危険度は1始まりで最小は1。
// 単調非減少で、同じ入力は常に同じ危険度を返す。
func dangerLevel(days int) int {
	if days < 0 {
		days = 0
	}
	return 1 + days/dangerDaysPerLevel
}

// DangerLevelAt は world から経過日数を引いて危険度を返す。
func DangerLevelAt(world w.World) int {
	return dangerLevel(GetGameTime(world).GetDayNumber())
}
