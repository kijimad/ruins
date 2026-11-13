package overworld

import (
	"fmt"
	"math/rand/v2"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// spawnFieldLoot は loot の item group から1山を抽選し、床へアイテムを spawn する。抽選は
// raw.SelectFromItemGroup がグループとコレクションを引き分け、pack を解釈する。地上フロアなので深度は扱わず
// group を直接引く。収納 loot の populateStorageLoot と同型で、建物ローカルの決定的 RNG を使い再訪で一致させる。
func spawnFieldLoot(world w.World, groupID string, pos consts.Coord[consts.Tile], rng *rand.Rand) error {
	draws, err := raw.SelectFromItemGroup(world.Resources.RawMaster, groupID, rng)
	if err != nil {
		return err
	}
	for _, d := range draws {
		if d.Name == "" {
			continue
		}
		if _, err := lifecycle.SpawnFieldItem(world, d.Name, pos.X, pos.Y, d.Count); err != nil {
			return err
		}
	}
	return nil
}

// spawnFacilityBeing は施設 fac の敵の表から危険度で1体を引き、建物の中の pos へ置く。危険度帯に該当が無ければ置かない。
func spawnFacilityBeing(world w.World, fac string, danger consts.Danger, pos consts.Coord[consts.Tile], rng *rand.Rand) error {
	table, err := urbanEnemyTableFor(world.Resources.RawMaster, fac)
	if err != nil {
		return err
	}
	name, err := raw.SelectEnemyByWeight(table, rng, int(danger))
	if err != nil {
		return err
	}
	if name == "" {
		return nil
	}
	_, err = lifecycle.SpawnEnemy(world, pos, name)
	return err
}

// populateStorageLoot は収納家具に、パレットが書いた戦利品表 tableID から件数ぶん重み抽選して入れる。
// ダンジョン生成の populateStorageLoot と同型で、overworld は建物ローカルの決定的 RNG を使う。施設 fac の
// 荒らされ率で空のまま残す。収納家具なのに表が無ければパレットの書き漏れとして error を返す。
func populateStorageLoot(world w.World, entity ecs.Entity, propName, tableID, fac string, rng *rand.Rand) error {
	propRaw, err := raw.GetProp(world.Resources.RawMaster, propName)
	if err != nil {
		return err
	}
	if propRaw.Storage == nil {
		return nil // 収納でない家具は何も入れない
	}
	if tableID == "" {
		return fmt.Errorf("storage furniture %q has no loot table in the room definition", propName)
	}
	looted, err := raw.FacilityLootedChance(world.Resources.RawMaster, fac)
	if err != nil {
		return err
	}
	if rng.IntN(100) < looted {
		return nil
	}
	itemTable, err := raw.GetItemTable(world.Resources.RawMaster, tableID)
	if err != nil {
		return err
	}
	// ルート数はダイス表記で決める。省略時は1個
	lootDice := consts.Dice{Base: 1, Sides: 1}
	if propRaw.Storage.LootCount != nil {
		d, err := consts.ParseDice(*propRaw.Storage.LootCount)
		if err != nil {
			return fmt.Errorf("invalid lootCount notation for storage '%s': %w", propName, err)
		}
		lootDice = d
	}
	// 危険度は北進度と経過日数の高い方。収納物のタイル位置を AbsChunkRow でチャンク行へ移して深度を引く
	danger := query.DangerLevelAt(world)
	if sb := query.GetSeamlessBand(world); sb != nil {
		depth := sb.DepthOfChunkRow(sb.AbsChunkRow(world.Components.GridElement.Get(entity).Y))
		if d := query.DangerLevelForDepth(depth); d > danger {
			danger = d
		}
	}
	n := lootDice.Roll(rng)
	for range n {
		draws, err := raw.DrawFromItemTable(world.Resources.RawMaster, itemTable, rng, int(danger))
		if err != nil {
			return err
		}
		for _, d := range draws {
			if _, err := lifecycle.SpawnStorageItem(world, d.Name, d.Count, entity); err != nil {
				return err
			}
		}
	}
	return nil
}
