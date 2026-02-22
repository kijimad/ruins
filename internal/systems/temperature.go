package systems

import (
	"errors"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TemperatureSystem は体温の更新を行うシステム
type TemperatureSystem struct{}

// String はシステム名を返す
func (sys TemperatureSystem) String() string {
	return "TemperatureSystem"
}

// 収束レート
// 半減期約30ターンで収束する
const convergenceRate = 0.03

// 快適温度（摂氏）
const comfortableTemp = 20

// Update は体温を更新する
func (sys *TemperatureSystem) Update(world w.World) error {
	dungeonRes := world.Resources.Dungeon
	if dungeonRes == nil {
		return errors.New("ダンジョンリソースが設定されていない")
	}

	// ダンジョン定義を取得
	def, ok := dungeon.GetDungeon(dungeonRes.DefinitionName)
	if !ok {
		return nil
	}

	// 環境気温を計算
	envTemp := calculateEnvironmentTemperature(def)

	// BodyTemperatureを持つエンティティを処理
	world.Manager.Join(
		world.Components.BodyTemperature,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		bt := world.Components.BodyTemperature.Get(entity).(*gc.BodyTemperature)
		updateBodyTemperature(bt, envTemp)
	}))

	return nil
}

// calculateEnvironmentTemperature は環境気温を計算する
func calculateEnvironmentTemperature(def dungeon.Definition) int {
	return def.BaseTemperature
}

// updateBodyTemperature は体温を更新する
func updateBodyTemperature(bt *gc.BodyTemperature, envTemp int) {
	for i := 0; i < int(gc.BodyPartCount); i++ {
		part := gc.BodyPart(i)
		state := &bt.Parts[i]

		// 収束温度を計算
		// 収束温度 = 50 + (環境気温 - 快適温度) * 2
		convergent := gc.TempNormal + (envTemp-comfortableTemp)*2
		state.Convergent = gc.ClampTemp(convergent)

		// 現在温度を収束温度に近づける
		delta := state.Convergent - state.Temp
		change := int(float64(delta) * convergenceRate)
		if change == 0 && delta != 0 {
			// 最低でも1は変化させる
			if delta > 0 {
				change = 1
			} else {
				change = -1
			}
		}
		state.Temp = gc.ClampTemp(state.Temp + change)

		// 凍傷タイマーを更新（手・足のみ）
		if gc.IsExtremity(part) {
			updateFrostbiteTimer(bt, part)
		}
	}
}

// updateFrostbiteTimer は凍傷タイマーを更新する
func updateFrostbiteTimer(bt *gc.BodyTemperature, part gc.BodyPart) {
	level := bt.GetLevel(part)
	state := &bt.Parts[part]

	switch level {
	case gc.TempLevelFreezing:
		state.FrostbiteTimer += 5 // 非常に危険
	case gc.TempLevelVeryCold:
		state.FrostbiteTimer += 3 // 危険
	case gc.TempLevelCold:
		state.FrostbiteTimer++ // リスク
	default:
		state.FrostbiteTimer -= 2 // 回復
	}

	// 範囲を制限
	if state.FrostbiteTimer < 0 {
		state.FrostbiteTimer = 0
	}
	if state.FrostbiteTimer > 100 {
		state.FrostbiteTimer = 100
	}

	// 凍傷発症判定
	if state.FrostbiteTimer >= 100 {
		state.HasFrostbite = true
	}
}
