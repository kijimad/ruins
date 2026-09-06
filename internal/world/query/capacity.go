package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む身体機能を返す。怪我・病気・疲労・空腹はすべて不調として
// HealthStatus に集約され、BodyFuncs の導出で意識を master 乗数として局所機能へ掛け込む。
// ここは HealthStatus を持たない対象へ健康な既定を返す薄いラッパ
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	if world.Components.HealthStatus.Has(entity) {
		return world.Components.HealthStatus.Get(entity).BodyFuncs()
	}
	return gc.HealthyBodyFuncs()
}
