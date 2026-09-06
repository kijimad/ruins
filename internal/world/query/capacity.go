package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// EffectiveBodyFuncs は消費側が読む最終的な身体機能を返す。全身性の低下をここ1箇所で各機能へ畳む。
// 全身性は不調由来で、痛み・全身の不調に加え疲労・空腹も Exhaustion・Malnutrition 不調として合流する。
// 速度・命中はこの値を読むだけでよく、意識を消費側で掛け直す必要はない。
// 部位ごとの怪我と全身性はすべて素の HealthStatus.BodyFuncs の意識へ集約済みで、ここは各機能へ配るだけ
func EffectiveBodyFuncs(world w.World, entity ecs.Entity) gc.BodyFuncs {
	raw := gc.HealthyBodyFuncs()
	if world.Components.HealthStatus.Has(entity) {
		raw = world.Components.HealthStatus.Get(entity).BodyFuncs()
	}
	// 全身性は 100 と素の意識の差。不調(痛み・全身の不調・疲労・空腹)がすべてここに畳まれている
	systemic := int(consts.PercentBase) - int(raw.Consciousness)
	sub := func(v consts.Percent) consts.Percent {
		return consts.Percent(max(int(v)-systemic, 0))
	}
	raw.Manipulation = sub(raw.Manipulation)
	raw.Moving = sub(raw.Moving)
	raw.Sight = sub(raw.Sight)
	return raw
}
