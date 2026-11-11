package activity

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// SleepConditions は actor の現在地で入眠できるかを判定するための条件をまとめた評価結果。
// 入眠可否は SleepBehavior.Validate と states のプロンプト起動が同じこの値を見るため、
// ゲートと表示がずれない。疲労・気温・安全のどれかを欠くと入眠できない。
type SleepConditions struct {
	Sleepy        bool // 疲労が眠れる段階にあるか。疲労を持たない者は眠れない
	TemperatureOK bool // 気温が入眠可能な帯に収まるか
	AreaSafe      bool // 周囲に敵対エンティティがいないか
}

// EvaluateSleepConditions は actor の現在地での睡眠条件を評価する。
// 気温と安全は query の純粋な読み取りを使い、判定を1箇所に集約する。
func EvaluateSleepConditions(world w.World, actor ecs.Entity) SleepConditions {
	sc := SleepConditions{
		AreaSafe:      query.IsAreaSafe(world, actor),
		TemperatureOK: true, // 座標が無い場所では気温では妨げない
	}

	if world.Components.Fatigue.Has(actor) {
		defs := world.Resources.NeedDefs
		stage, _ := defs.FatigueStageOf(world.Components.Fatigue.Get(actor).GetLevel(defs))
		sc.Sleepy = stage.CanSleep
	}

	if world.Components.GridElement.Has(actor) {
		grid := world.Components.GridElement.Get(actor)
		lower, upper := query.SleepableTemperatureRange(world, actor)
		if ambient, err := query.AmbientTemperatureAt(world, grid.X, grid.Y); err == nil {
			sc.TemperatureOK = ambient >= lower && ambient <= upper
		}
	}

	return sc
}

// TooTired は疲労が足りず眠れない状態かを返す。眠れない段階か、疲労を持たない者は眠れない。
// 寝すぎを防ぐため一定の疲労がないと入眠できない。
func (sc SleepConditions) TooTired() bool {
	return !sc.Sleepy
}

// CanSleep は入眠を妨げるブロック条件が一つも無いかを返す。
func (sc SleepConditions) CanSleep() bool {
	return !sc.TooTired() && sc.TemperatureOK && sc.AreaSafe
}
