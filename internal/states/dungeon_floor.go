package states

import (
	"fmt"
	"math/rand/v2"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/mlange-42/ark/ecs"
)

// ダンジョンのフロア生成・階層遷移・ポータル配線を dungeon.go から分離する。
// DungeonState のメソッドはこのファイルにも置く。

// resolveDungeonKind は名前から通常ダンジョン種別を引く。未登録、またはオーバーワールドのような
// フロアを生成しない種別なら error を返す。フロア生成の入口を1箇所に集約する。
func resolveDungeonKind(defName string) (*dungeon.DungeonKind, error) {
	kind, found := dungeon.GetStageKind(defName)
	if !found {
		return nil, fmt.Errorf("ステージ定義が見つかりません: %s", defName)
	}
	dk, ok := kind.(*dungeon.DungeonKind)
	if !ok {
		return nil, fmt.Errorf("フロア生成できないステージ種別です: %s", defName)
	}
	return dk, nil
}

// dungeonStageKey は遺跡名と深度でダンジョン階のステージキーを返す。
// 遺跡はオーバーワールドの入口から名前付きで入り、複数の遺跡が同一 world に共存しうる。
// よって階のキーは遺跡名で区別する必要がある。enterDungeon が焼く入口(1階)のキーと
// descend が作る深い階のキーをこの関数で揃え、上り階段の結線が正しい階を指すようにする。
func dungeonStageKey(defName string, depth int) gc.StageKey {
	return gc.NewNamedDungeonStage(defName, depth)
}

// spawnFloor は depth のフロアを生成して world に配置し、生成物に StageBound を付ける。
// プレイヤー開始位置と、開始位置に置いた上り階段エンティティを返す。上り階段には呼び出し側が
// 戻り先を結線する。プレイヤー配置・探索リセット・現ステージ更新は呼び出し側が行う
func (st *DungeonState) spawnFloor(world w.World, depth int, def *dungeon.DungeonKind, key gc.StageKey) (consts.Coord[consts.Tile], ecs.Entity, error) {
	var zero consts.Coord[consts.Tile]
	var noEntity ecs.Entity

	stageSeed := world.Config.RNG.Uint64()
	stageRNG := rand.New(rand.NewPCG(stageSeed, 0))

	// ビルダータイプを決定する。最終階層かつボスフロアプランナーがあればボスフロアにする
	var builderType mapplanner.PlannerType
	switch bossPlanner, isBoss := def.BossPlanner(depth); {
	case isBoss:
		builderType = bossPlanner
	case st.BuilderType.PlannerFunc == nil || st.BuilderType.Name == mapplanner.PlannerTypeRandom.Name:
		// BuilderType 未設定(オーバーワールドから遺跡へ入った State は帯用で BuilderType を
		// 持たない)か Random なら、定義のプランナープールから選ぶ。ゼロ値をそのまま使うと
		// PlannerFunc が nil で生成が panic する
		var err error
		builderType, err = def.SelectPlanner(stageRNG)
		if err != nil {
			return zero, noEntity, err
		}
	default:
		builderType = st.BuilderType
	}

	// テーブル名と階層をプランナーに渡す。エントリの解決はプランナーが行う
	builderType.EnemyTableName = def.EnemyTableName()
	builderType.ItemTableName = def.ItemTableName()
	builderType.Depth = depth

	plan, err := mapplanner.Plan(world, consts.MapTileWidth, consts.MapTileHeight, stageSeed, builderType)
	if err != nil {
		return zero, noEntity, err
	}
	level, err := mapspawner.Spawn(world, plan)
	if err != nil {
		return zero, noEntity, err
	}
	// フィールド寸法をこの階のStageFieldへ記録する。生成物と同じ明示 key に束縛するため、
	// SwapTo が CurrentStage を最後に更新する順序に依存しない
	query.EnsureStageField(world, key).Level = level

	start, err := plan.GetPlayerStartPosition()
	if err != nil {
		return zero, noEntity, err
	}

	// 上り階段を開始位置に置く。降りてきた場所が、上りで戻ってくる場所になる。
	// 最上階(floor 1)では上り階段がダンジョン脱出口を兼ねる。町(depth 0)には置かない
	var upStair ecs.Entity
	if depth > 0 {
		e, err := lifecycle.SpawnProp(world, "warp_prev", start.X, start.Y)
		if err != nil {
			return zero, noEntity, err
		}
		upStair = e
	}

	// 生成物(上り階段を含む)をこのステージへ束縛して識別できるようにする
	stage.Bind(world, key)

	return start, upStair, nil
}

// descend は1つ下の階へ swapTo で移動する。現階を退避し、未訪問なら生成、訪問済みなら再稼働する。
// TransPush で新ステートを積むのでなく、同一 State 内で現階と入れ替えるのが共存方式の要点
func (st *DungeonState) descend(world w.World) error {
	// 今いる遺跡の定義名は現ステージのキーから導出する
	defName := query.GetDungeon(world).CurrentStage.Name
	fromStage := dungeonStageKey(defName, st.Depth)
	// 現階の下り階段の位置。生成する階の上り階段の戻り先として結線する
	fromDownStairPos, hasDownStair := findPortalPosition(world, gc.InteractionPortalNext)

	nextDepth := st.Depth + 1
	target := dungeonStageKey(defName, nextDepth)

	// 生成は swapTo の callback で行う。未訪問のときだけ呼ばれる。
	// def 参照も生成時だけに閉じ、訪問済みの再稼働では不要にする
	var playerPos consts.Coord[consts.Tile]
	var generated bool
	if err := stage.SwapTo(world, target, func(world w.World, key gc.StageKey) error {
		def, err := resolveDungeonKind(defName)
		if err != nil {
			return err
		}
		start, upStair, err := st.spawnFloor(world, nextDepth, def, key)
		if err != nil {
			return err
		}
		// 生成した階の上り階段に、降りてきた元階の下り階段への戻り先を焼く。
		// これで ascend は探索なしに戻り先ステージと座標を引ける
		if hasDownStair {
			if cerr := setPortalConnection(world, upStair, fromStage, fromDownStairPos); cerr != nil {
				return cerr
			}
		}
		playerPos = start
		generated = true
		return nil
	}); err != nil {
		return err
	}

	st.Depth = nextDepth

	// 生成フロアは開始位置(＝上り階段の位置)へ。訪問済みフロアの再訪は
	// そのフロアの上り階段、すなわち降りてくる側の位置へ戻す
	if generated {
		return lifecycle.MovePlayerToPosition(world, playerPos)
	}
	pos, ok := findPortalPosition(world, gc.InteractionPortalPrev)
	if !ok {
		// 訪問済みの階には必ず上り階段があるはず。無ければステージ切替済みで
		// プレイヤーが元座標に取り残されるので、silent にせず error にする
		return fmt.Errorf("再訪した階に上り階段が見つかりません: 深度%d", nextDepth)
	}
	return lifecycle.MovePlayerToPosition(world, pos)
}

// findPortal は現ステージの指定種別ポータルのエンティティと位置を返す。
// 退避中ステージのポータルは ActiveFilter で除外される。先着1件を採用するが、途中 return せず
// 反復は最後まで続ける。Ark のワールドロックを外すため。実ゲームでは各ステージにポータルは
// 1つなので先着で一意に定まる
func findPortal(world w.World, kind gc.InteractionKind) (ecs.Entity, consts.Coord[consts.Tile], bool) {
	var found ecs.Entity
	var pos consts.Coord[consts.Tile]
	ok := false
	q := query.ActiveFilter2[gc.Interactable, gc.GridElement](world).Query()
	for q.Next() {
		e := q.Entity()
		if !ok && slices.Contains(world.Components.Interactable.Get(e).Interactions, kind) {
			found = e
			pos = world.Components.GridElement.Get(e).Coord
			ok = true
		}
	}
	return found, pos, ok
}

// findPortalPosition は findPortal のうち位置だけを返す薄いラッパー。
func findPortalPosition(world w.World, kind gc.InteractionKind) (consts.Coord[consts.Tile], bool) {
	_, pos, ok := findPortal(world, kind)
	return pos, ok
}

// setPortalConnection はポータルに行き先ステージと着地座標を結線する。
// 生成時に両端を結線し、以降の往復は探索でなくこの結線から行き先を引く。
func setPortalConnection(world w.World, portal ecs.Entity, target gc.StageKey, coord consts.Coord[consts.Tile]) error {
	return gc.Upsert(world.ECS, world.Components.PortalConnection, portal, &gc.PortalConnection{Stage: target, Coord: coord})
}

// ascend は現階の上り階段の結線した戻り先へ swapTo で移動する。上り先は訪問済み前提で再稼働する。
// 戻り先ステージと着地座標は生成時に上り階段へ結線済みなので、探索でなく結線から引く。
// 上り階段の結線があれば移動して true を返す。結線が無い、たとえば最上階の脱出口なら false を
// 返し、街やオーバーワールドへの脱出は呼び出し側が扱う。上り先が浅い階でも遺跡→地上でも同一機構。
func (st *DungeonState) ascend(world w.World) (bool, error) {
	// 現階の上り階段。生成時に戻り先が結線されている
	upStair, _, ok := findPortal(world, gc.InteractionPortalPrev)
	if !ok {
		return false, nil
	}
	if !world.Components.PortalConnection.Has(upStair) {
		// 結線なし。最上階の脱出口。呼び出し側が脱出を扱う
		return false, nil
	}
	// 行き先を値でコピーする。swapTo が Suspended を付けてアーキタイプが変わると
	// コンポーネントポインタは無効化されるため、構造変更の前に取り出す
	conn := *world.Components.PortalConnection.Get(upStair)
	target := conn.Stage

	// 上り先は訪問済み前提。未訪問なら生成でなくエラーにする
	if err := stage.SwapTo(world, target, func(_ w.World, _ gc.StageKey) error {
		return fmt.Errorf("上り先の階が存在しません: %+v", target)
	}); err != nil {
		return false, err
	}

	st.Depth = target.Depth
	// SwapTo 後は現ステージ=target なので現在地で判定する
	if query.IsOnOverworld(world) {
		// 地上の StageField が resume で帯寸法の Level を戻すため寸法の手復元は不要。視界だけ強制再計算する
		query.GetVisionState(world).NeedsForceUpdate = true
	}

	if err := lifecycle.MovePlayerToPosition(world, conn.Coord); err != nil {
		return false, err
	}
	return true, nil
}

// enterDungeon はオーバーワールドから遺跡へ入る。現在地(入口座標)を上り階段へ結線して戻れるようにする。
// descend の遺跡版で、行き先が1つ深い階でなく遺跡1階になる。
func (st *DungeonState) enterDungeon(world w.World, defName string) error {
	fromStage := query.GetDungeon(world).CurrentStage
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	// 入口のオーバーワールド座標。swapTo 前に値でコピーする
	fromPos := world.Components.GridElement.Get(player).Coord

	target := gc.NewNamedDungeonStage(defName, 1)

	var landing consts.Coord[consts.Tile]
	var generated bool
	if err := stage.SwapTo(world, target, func(world w.World, key gc.StageKey) error {
		def, derr := resolveDungeonKind(defName)
		if derr != nil {
			return derr
		}
		start, upStair, serr := st.spawnFloor(world, 1, def, key)
		if serr != nil {
			return serr
		}
		// 遺跡の上り階段(=出口)に、入ってきたオーバーワールドの入口座標を結線する。
		// これで exit は入った入口へ正確に戻れる。入口が複数でも曖昧にならない
		if cerr := setPortalConnection(world, upStair, fromStage, fromPos); cerr != nil {
			return cerr
		}
		landing = start
		generated = true
		return nil
	}); err != nil {
		return err
	}

	st.Depth = 1
	// 現ステージ=遺跡1階のキーが定義名を持つため、別途 DefinitionName を記録する必要はない

	if generated {
		return lifecycle.MovePlayerToPosition(world, landing)
	}
	// 再訪。遺跡の上り階段(入口)へ戻す。訪問済みなら必ず存在するはず。
	// 無ければプレイヤーが元座標に取り残されるので silent にせず error にする
	pos, ok := findPortalPosition(world, gc.InteractionPortalPrev)
	if !ok {
		return fmt.Errorf("再訪した遺跡に上り階段が見つかりません: %s", defName)
	}
	return lifecycle.MovePlayerToPosition(world, pos)
}
