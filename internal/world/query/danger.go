package query

import (
	w "github.com/kijimaD/ruins/internal/world"
)

// dangerDaysPerLevel は危険度1段あたりの経過日数。値は暫定で、実プレイの伸びを測って振り直す。
const dangerDaysPerLevel = 3

// DangerLevel は経過日数から危険度を返す純関数。
// 単調非減少で、同じ入力は常に同じ危険度を返す。
// ruins は留まれない世界で、時間の経過が前進とほぼ等しい。距離を別軸に足さず、
// 時間だけで危険度を出す。前進の空間勾配は、時間と移動が結合することで体感として立つ。
func DangerLevel(days int) int {
	if days < 0 {
		days = 0
	}
	return days / dangerDaysPerLevel
}

// DangerLevelAt は world から経過日数を引いて DangerLevel を呼ぶ。
func DangerLevelAt(world w.World) int {
	return DangerLevel(GetGameTime(world).GetDayNumber())
}
