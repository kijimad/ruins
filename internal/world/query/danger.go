package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
)

// 危険度の配合比。距離が主、日数が従。値は暫定で、実プレイの伸びを測って振り直す。
const (
	// dangerTilesPerLevel は危険度1段あたりの前進タイル数
	dangerTilesPerLevel = 50
	// dangerDaysPerLevel は危険度1段あたりの経過日数
	dangerDaysPerLevel = 3
)

// DangerLevel は前進距離と経過日数から危険度を返す純関数。
// 距離が主、日数が従。単調非減少で、同じ入力は常に同じ危険度を返す。
func DangerLevel(dist consts.AbsTileX, days int) int {
	if dist < 0 {
		dist = 0
	}
	if days < 0 {
		days = 0
	}
	return int(dist)/dangerTilesPerLevel + days/dangerDaysPerLevel
}

// DangerLevelAt は world から経過日数を引き、絶対 X を前進距離として DangerLevel を呼ぶ。
// absX は世界原点からの絶対 X で、前進距離そのものにあたる。
func DangerLevelAt(world w.World, absX consts.AbsTileX) int {
	return DangerLevel(absX, GetGameTime(world).GetDayNumber())
}
