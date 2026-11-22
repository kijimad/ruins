package gameaction

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// illnessOnsetTimer は発症したての病気の初期進行度。発症する 25 以上にして浅く始め、早期発見を効かせる
const illnessOnsetTimer = 30

// ContractIllness は病気を1つ発症させる。病気は全身性なので胴に付ける。同じ病気が発症中なら重ねない。
// 治りかけで発症していない同じ病気が残っていれば、発症したての進行度に戻す。
// HealthStatus を持たない対象には何もしない。発症したら true を返し、身体機能の再計算を促す。
// 発症したての浅い進行度で始まり、以降の進行は不調ごとの回復モードに従う
func ContractIllness(world w.World, entity ecs.Entity, ct gc.ConditionType) bool {
	if !world.Components.HealthStatus.Has(entity) {
		return false
	}
	part := &world.Components.HealthStatus.Get(entity).Parts[gc.BodyPartTorso]
	if existing := part.GetCondition(ct); existing != nil && existing.IsActive() {
		return false
	}
	part.SetCondition(gc.HealthCondition{
		Type:     ct,
		Timer:    illnessOnsetTimer,
		Severity: gc.TimerToSeverity(illnessOnsetTimer),
	})
	if !world.Components.StatsChanged.Has(entity) {
		world.Components.StatsChanged.Add(entity, &gc.StatsChanged{})
	}
	if world.Components.Player.Has(entity) {
		name := query.ConditionName(world, ct)
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "You contracted %s.", name)).
			Log()
	}
	return true
}
