package balance

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/activity"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// DriveRangeTiles は所持燃料 fuel と積載重量 load でキューブが運転できるタイル数を返す。query.DriveFuelCost を
// そのまま使い燃費式に自動追従する。積荷が重いほど1タイルのコストが上がり航続が縮む重量↔移動のトレード。
func DriveRangeTiles(fuel consts.Heat, load consts.Milligram) float64 {
	cost := query.DriveFuelCost(load)
	if cost <= 0 {
		return 0
	}
	return float64(fuel) / float64(cost)
}

// DriveRangeAllFuel は容量いっぱいを material の燃料で満たしたときの航続タイルを返す。
// 燃料自身が積載重量になるので、積むほど燃費が悪化する自己ブレーキを含む。
func DriveRangeAllFuel(material oapi.Material, capacityKg int) float64 {
	weight := consts.Milligram(capacityKg) * consts.MilligramPerKg
	fuel := query.HeatOf(material, weight)
	return DriveRangeTiles(fuel, weight)
}

// FuelBurnTurns は material を weightKg だけ地面直の火にくべたとき増える燃焼ターン数を返す。熱量を地面の燃焼効率で
// 割り引く。query.HeatOf と consts.Heat.BurnTurns に自動追従する。移動燃料と同じ熱量が火では燃焼時間へ変わる。
func FuelBurnTurns(material oapi.Material, weightKg int) int {
	weight := consts.Milligram(weightKg) * consts.MilligramPerKg
	return int(query.HeatOf(material, weight).BurnTurns(query.GroundBurnEfficiency))
}

// TreeFuel は木1本から燃料を得る収支。素手で殴り倒す経路と工具で伐る経路を並べる
type TreeFuel struct {
	HandTurns     float64 // 素手で殴り倒す期待打数
	HandBurnTurns float64 // 殴り倒して落ちる燃料を地面の火にくべた期待燃焼ターン
	ToolTurns     float64 // 工具で伐る作業ターン。スキル0とする
	ToolBurnTurns float64 // 工具で伐って得る燃料の期待燃焼ターン
}

// TreeFuelBalance は木 treeID を素手と grade の工具で伐ったときの収支を返す。打数は ExpectedTTK、
// 作業ターンは activity.RequiredDisassemblyAP、産出は lifecycle.ExpectedDisassemblyYields に追従する
func TreeFuelBalance(master oapi.Raws, player CombatantStats, bareHands WeaponStats, treeID string, toolGrade int) (TreeFuel, error) {
	prop, err := raw.GetProp(master, treeID)
	if err != nil {
		return TreeFuel{}, err
	}
	if prop.Disassembly == nil {
		return TreeFuel{}, fmt.Errorf("prop %q has no disassembly definition", treeID)
	}
	def := prop.Disassembly

	handBurn, err := expectedYieldBurnTurns(master, def, 0, false)
	if err != nil {
		return TreeFuel{}, err
	}
	toolBurn, err := expectedYieldBurnTurns(master, def, toolGrade, true)
	if err != nil {
		return TreeFuel{}, err
	}
	return TreeFuel{
		HandTurns:     ExpectedTTK(player, CombatantStats{HP: prop.Hp}, bareHands),
		HandBurnTurns: handBurn,
		ToolTurns:     float64(activity.RequiredDisassemblyAP(def.BaseAP, 0, toolGrade)) / consts.StandardActionCost,
		ToolBurnTurns: toolBurn,
	}, nil
}

// expectedYieldBurnTurns は分解産出の期待個数を地面の火での燃焼ターンへ換算して合計する
func expectedYieldBurnTurns(master oapi.Raws, def *oapi.Disassembly, toolGrade int, full bool) (float64, error) {
	yields, err := lifecycle.ExpectedDisassemblyYields(def, 0, toolGrade, full)
	if err != nil {
		return 0, err
	}
	total := 0.0
	for id, count := range yields {
		item, err := raw.FindItem(master, id)
		if err != nil {
			return 0, err
		}
		if item.Material == nil || item.Weight == nil {
			continue
		}
		weight, err := consts.ParseWeight(*item.Weight)
		if err != nil {
			return 0, err
		}
		total += count * float64(query.HeatOf(*item.Material, weight).BurnTurns(query.GroundBurnEfficiency))
	}
	return total, nil
}
