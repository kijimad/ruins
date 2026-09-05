package systems

import (
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// SleepConditions は actor の現在地で睡眠する際の各条件をまとめた評価結果。
// 疲労・気温・安全は入眠を妨げうるブロック条件で、寝具は回復効率に効くだけの情報。
// 表示は states 層が行い、ここでは判定を1箇所に集約する。プロンプト表示と入眠可否が
// 同じ値を見るため、表示と実際のゲートがずれない。
type SleepConditions struct {
	HasFatigue     bool            // 疲労コンポーネントを持つか。持たない者は眠れない
	Fatigue        gc.FatigueLevel // 疲労段階
	HasAmbient     bool            // 座標があり気温を評価できたか
	Ambient        int             // 現在地に適用される気温
	SleepableLower int             // 入眠可能な気温の下限。HasAmbient のとき意味を持つ
	SleepableUpper int             // 入眠可能な気温の上限
	TemperatureOK  bool            // 気温が入眠可能な帯に収まるか
	BeddingQuality consts.Percent  // 寝具の質。PercentBase が地べたの基準
	AreaSafe       bool            // 周囲に敵対エンティティがいないか
}

// EvaluateSleepConditions は actor の現在地での睡眠条件を評価する。
func EvaluateSleepConditions(world w.World, actor ecs.Entity) SleepConditions {
	sc := SleepConditions{
		AreaSafe:       activity.IsAreaSafe(actor, world),
		BeddingQuality: activity.BeddingQualityAt(actor, world),
		TemperatureOK:  true, // 座標が無い場所では気温では妨げない
	}

	if world.Components.Fatigue.Has(actor) {
		sc.HasFatigue = true
		sc.Fatigue = world.Components.Fatigue.Get(actor).GetLevel()
	}

	if world.Components.GridElement.Has(actor) {
		grid := world.Components.GridElement.Get(actor)
		sc.SleepableLower, sc.SleepableUpper = SleepableTemperatureRange(world, actor)
		if ambient, err := AmbientTemperatureAt(world, grid.X, grid.Y); err == nil {
			sc.HasAmbient = true
			sc.Ambient = ambient
			sc.TemperatureOK = ambient >= sc.SleepableLower && ambient <= sc.SleepableUpper
		}
	}

	return sc
}

// TooTired は疲労が足りず眠れない状態かを返す。快調 Rested か疲労を持たない者は眠れない。
// 寝すぎを防ぐため一定の疲労がないと入眠できない。
func (sc SleepConditions) TooTired() bool {
	return !sc.HasFatigue || sc.Fatigue == gc.FatigueRested
}

// CanSleep は入眠を妨げるブロック条件が一つも無いかを返す。寝具は効率に効くだけで妨げない。
func (sc SleepConditions) CanSleep() bool {
	return !sc.TooTired() && sc.TemperatureOK && sc.AreaSafe
}
