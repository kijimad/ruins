package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/stage"
)

// townNPCs は街に配置する会話NPCの定義名と、街中心からの相対座標。
// 会話 InteractionTalk で店(商人)・雇用(酒場の主人)・合成(怪しい科学者)を開く。
// 機能は既存の「NPC会話→ダイアログ→メニュー」機構をそのまま使い、街を専用ステージから
// オーバーワールドの地物へ移すことだけを行う。
var townNPCs = []struct {
	name string
	dx   consts.Tile
	dy   consts.Tile
}{
	{"商人", -2, -1},
	{"酒場の主人", -2, 1},
	{"怪しい科学者", -4, 0},
}

// townStorageProp は街に置く収納propの raw 名。中身のない倉庫としてプレイヤーが預けるのに使う。
const townStorageProp = "木箱"

// spawnTown はオーバーワールド開始チャンクに街を構成する。center を街の中心として、
// 会話NPC(店・雇用・合成)と収納propを近傍へ決定的に配置し、オーバーワールド帯へ束縛する。
//
// 街はステージでなくオーバーワールドの地物なので、専用の State を持たず prop として常在する。
// これで新規ゲームの開始点を「街を含むオーバーワールド」にでき、TownState への遷移が不要になる。
func spawnTown(world w.World, center consts.Coord[consts.Tile]) error {
	for _, n := range townNPCs {
		pos := consts.Coord[consts.Tile]{X: center.X + n.dx, Y: center.Y + n.dy}
		if _, err := lifecycle.SpawnNeutralNPC(world, pos, n.name); err != nil {
			return fmt.Errorf("街NPCの配置に失敗 (%s): %w", n.name, err)
		}
	}

	// 収納。中身のない倉庫として置き、相互作用 InteractionStorage で開ける
	if _, err := lifecycle.SpawnProp(world, townStorageProp, center.X-4, center.Y-2); err != nil {
		return fmt.Errorf("街の収納の配置に失敗: %w", err)
	}

	// 置いた街エンティティ(未束縛)をオーバーワールド帯へ束縛する。帯タイル・遺跡入口は
	// 既に StageBound を持ち、プレイヤーは除外されるので、街の NPC/prop だけが束縛される。
	// これで遺跡進入時に街も帯とともに退避され、戻ると復元される
	stage.Bind(world, gc.NewOverworldStage())
	return nil
}
