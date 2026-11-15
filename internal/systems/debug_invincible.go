package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// DebugInvincibleSystem はデバッグの無敵の間、プレイヤーの HP・体温・空腹・疲労を満たし、怪我や病気の不調を消し、倒れていれば起こす。
// 死の判定より前に毎フレーム走るので、致死の一撃を受けても死なない
type DebugInvincibleSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装
func (sys DebugInvincibleSystem) String() string {
	return "DebugInvincibleSystem"
}

// Update はデバッグの無敵の間、プレイヤーを満たす
// w.Updater interfaceを実装
func (sys *DebugInvincibleSystem) Update(world w.World) error {
	cfg := world.Resources.Config
	if cfg == nil || !cfg.DebugInvincible {
		return nil
	}
	var players []ecs.Entity
	q := ecs.NewFilter1[gc.Player](world.ECS).Query()
	for q.Next() {
		players = append(players, q.Entity())
	}
	for _, p := range players {
		if world.Components.Dead.Has(p) {
			world.Components.Dead.Remove(p)
		}
		if world.Components.HP.Has(p) {
			hp := world.Components.HP.Get(p)
			hp.Current = hp.Max
		}
		if world.Components.HealthStatus.Has(p) {
			hs := world.Components.HealthStatus.Get(p)
			hs.BodyTempOffset = 0
			healed := false
			for i := range hs.Parts {
				if len(hs.Parts[i].Conditions) > 0 {
					hs.Parts[i].Conditions = nil
					healed = true
				}
			}
			// 不調は能力値を下げるので、消したら能力値を計算し直させる
			if healed && !world.Components.StatsChanged.Has(p) {
				world.Components.StatsChanged.Add(p, &gc.StatsChanged{})
			}
		}
		if world.Components.Hunger.Has(p) {
			hunger := world.Components.Hunger.Get(p)
			hunger.Current = hunger.Max
		}
		if world.Components.Fatigue.Has(p) {
			world.Components.Fatigue.Get(p).Current = 0
		}
	}
	return nil
}
