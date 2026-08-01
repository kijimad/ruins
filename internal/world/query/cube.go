package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// CubeWeight は内部ステージのフィールド上にある全 gc.Weight 保有エンティティの重量総和を返す。
// item も、暖や作業の効果を持つ prop も、内部の床へ置けば等しく重量になる。
// 所有者ベースの calculateOwnedWeight とは別軸で、StageBound.Key で集計する。
//
// LocationOnField を条件に含め、床にある物だけを数える。拾って背包へ移すと LocationOnField が
// 外れて総重量から抜ける。StageBound は拾っても残るため、これが無いと持ち去った物まで数えてしまう。
//
// 内部は外にいる間 Suspended になるが、総重量は退避中も保持したい。Suspended は退避で付くが
// LocationOnField は残るので、Suspended を除外する ActiveFilter でなく生のフィルタで全ステージを
// 走査し、Key 一致で絞る。
func CubeWeight(world w.World, interior gc.StageKey) consts.Milligram {
	var total consts.Milligram
	weightQuery := ecs.NewFilter2[gc.Weight, gc.LocationOnField](world.ECS).Query()
	for weightQuery.Next() {
		entity := weightQuery.Entity()
		if world.Components.StageBound.Has(entity) && world.Components.StageBound.Get(entity).Key == interior {
			total += GetEntityWeight(world, entity)
		}
	}
	return total
}

// PushCost は総重量から1タイル押すのに要するAPを返す。空のキューブでも歩行の10倍規模の
// 基準がかかり、総重量に比例して増える。仲間や強化ではこの値は変わらず、変わるのは
// パーティAPで何ターンで払えるかである。
func PushCost(total consts.Milligram) int {
	kg := int(total / consts.MilligramPerKg)
	return consts.PushCostBase + consts.PushCostPerKg*kg
}

// PartyPushPower はこのターン押しへ充てられるAP総量を返す。Player と SquadMember の
// TurnBased.AP.Current を合算する。仲間を増やし強化すると増え、同じ PushCost を
// より少ないターンで払えて速く進む。押しコスト自体は不変。
func PartyPushPower(world w.World) int {
	var total int
	playerQuery := ActiveFilter2[gc.TurnBased, gc.Player](world).Query()
	for playerQuery.Next() {
		total += world.Components.TurnBased.Get(playerQuery.Entity()).AP.Current
	}
	memberQuery := ActiveFilter2[gc.TurnBased, gc.SquadMember](world).Query()
	for memberQuery.Next() {
		total += world.Components.TurnBased.Get(memberQuery.Entity()).AP.Current
	}
	return total
}
