package states

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/config"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/input"
	"github.com/kijimaD/ruins/internal/inputmapper"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/mapspawner"
	"github.com/kijimaD/ruins/internal/messagedata"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/mlange-42/ark/ecs"
)

// DungeonState はダンジョン探索中のゲームステート
type DungeonState struct {
	es.BaseState[w.World]
	// baseImage は下に敷く背景
	baseImage *ebiten.Image
	Depth     int
	// BuilderType は使用するマップビルダーのタイプ（BuilderTypeRandom の場合はランダム選択）
	BuilderType mapplanner.PlannerType
	// DefinitionName はダンジョン定義名。設定されていればOnStartでリソースに反映する
	DefinitionName string
	// Resume はセーブからの復帰モード。trueならマップ再生成とプレイヤー再配置を行わず、
	// 復元済みのワールド（地形・エンティティ・プレイヤー位置）をそのまま使う
	Resume bool

	// planner・newGame・session はオーバーワールドモード(定義が Seamless)のときだけ使う。
	// 帯固有のロジックは overworld.Session に閉じ込め、DungeonState は保持と委譲だけ行う
	planner mapplanner.PlannerType
	newGame *overworld.NewGameParams // 新規開始の帯パラメータ。ロード復元では nil
	session *overworld.Session       // OnStart で構成する帯セッション。通常ダンジョンでは nil
}

// isSeamless はこの State がオーバーワールド帯モードかを返す。オーバーワールドとダンジョンの
// 本質的な違いは帯の有無だけで、それはダンジョン定義の Seamless が表す。
func (st DungeonState) isSeamless() bool {
	def, ok := dungeon.GetDungeon(st.DefinitionName)
	return ok && def.Seamless
}

// NewOverworldState はオーバーワールド探索ステートのファクトリを返す。
//
// オーバーワールドは「帯を持つダンジョン」(DungeonOverworld, Seamless=true)で、専用の State 型は
// 持たず DungeonState として動く。帯固有のロジックは overworld.Session に閉じ込めてあり、
// DungeonState は OnStart でセッションを構成して開始を委譲し、Update でシフトを委譲するだけ。
//
// params が非 nil なら新規開始として初期帯を生成する。nil ならセーブからの復元とみなし、
// 帯パラメータは Session の Start がオーバーワールドのメタの SeamlessBand から読み取って再構築する。
func NewOverworldState(planner mapplanner.PlannerType, params *overworld.NewGameParams) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &DungeonState{
			// 定義名を Seamless なオーバーワールド定義にすることで、OnStart が帯モードへ分岐する
			DefinitionName: dungeon.DungeonOverworld.Name,
			planner:        planner,
			newGame:        params,
		}, nil
	}
}

// State interface ================

var _ es.State[w.World] = &DungeonState{}
var _ es.ActionHandler[w.World] = &DungeonState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *DungeonState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *DungeonState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *DungeonState) OnStart(world w.World) error {
	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height
	if screenWidth > 0 && screenHeight > 0 {
		st.baseImage = ebiten.NewImage(screenWidth, screenHeight)
		st.baseImage.Fill(theme.ScreenBackground)
	}

	// Seamless なオーバーワールドは帯セッションを構成して委譲する。帯固有のロジックは
	// overworld.Session に閉じ込め、DungeonState はここで開始を委譲するだけにする
	if st.isSeamless() {
		st.session = overworld.NewSession(st.planner, st.newGame)
		return st.session.Start(world)
	}

	// 設定されていればリソースに反映する
	if st.DefinitionName != "" {
		query.GetDungeon(world).DefinitionName = st.DefinitionName
	}
	// ダンジョン定義を取得する
	def, found := dungeon.GetDungeon(query.GetDungeon(world).DefinitionName)
	if !found {
		return fmt.Errorf("ダンジョン定義が見つかりません: %s", query.GetDungeon(world).DefinitionName)
	}
	// 復帰モードでは再生成せず、復元済みの地形・エンティティ・プレイヤー位置をそのまま使う
	if !st.Resume {
		key := dungeonStageKey(query.GetDungeon(world).DefinitionName, st.Depth)
		playerPos, _, err := st.spawnFloor(world, st.Depth, def, key)
		if err != nil {
			return err
		}
		// プレイヤーを配置する
		if err := lifecycle.MovePlayerToPosition(world, playerPos); err != nil {
			return err
		}
		// フロア移動時に探索済みマップをリセットし、現ステージを確定する
		stage.ResetExploredTiles(world)
		query.GetDungeon(world).CurrentStage = key
	}

	// 前フロア・復元前のSpatialIndexが残っている可能性があるため無効化して作り直す。
	// SpatialIndexはTurnPhaseEndでのみ無効化されるが、フロア遷移はTurnPhasePlayer中に
	// 発生するため、古いデータが残り移動不能になることがある
	query.InvalidateSpatialIndex(world)

	// ダンジョンタイトルエフェクト用エンティティを作成する
	screenW, screenH := world.Resources.GetScreenDimensions()
	titleText := def.Name
	if st.Depth > 0 {
		titleText = fmt.Sprintf("%s %dF", def.Name, st.Depth)
	}
	splashFace := world.Resources.UIResources.Text.SplashFontFace
	titleEffect := gc.NewSplashTextEffect(titleText, splashFace, screenW, screenH)
	titleEntity := world.ECS.NewEntity()
	world.Components.VisualEffects.Add(titleEntity, &gc.VisualEffects{
		Effects: []gc.VisualEffect{titleEffect},
	})

	return nil
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
func (st *DungeonState) spawnFloor(world w.World, depth int, def dungeon.Definition, key gc.StageKey) (consts.Coord[consts.Tile], ecs.Entity, error) {
	var zero consts.Coord[consts.Tile]
	var noEntity ecs.Entity

	stageSeed := world.Config.RNG.Uint64()
	stageRNG := rand.New(rand.NewPCG(stageSeed, 0))

	// ビルダータイプを決定する。最終階層かつBossPlannerTypeがあればボスフロアにする
	var builderType mapplanner.PlannerType
	switch {
	case def.BossPlannerType != nil && depth == def.TotalFloors:
		builderType = *def.BossPlannerType
	case st.BuilderType.PlannerFunc == nil || st.BuilderType.Name == mapplanner.PlannerTypeRandom.Name:
		// BuilderType 未設定(オーバーワールドから遺跡へ入った State は帯用で BuilderType を
		// 持たない)か Random なら、定義のプランナープールから選ぶ。ゼロ値をそのまま使うと
		// PlannerFunc が nil で生成が panic する
		var err error
		builderType, err = dungeon.SelectPlanner(def, stageRNG)
		if err != nil {
			return zero, noEntity, err
		}
	default:
		builderType = st.BuilderType
	}

	// テーブル名と階層をプランナーに渡す。エントリの解決はプランナーが行う
	builderType.EnemyTableName = def.EnemyTableName
	builderType.ItemTableName = def.ItemTableName
	builderType.Depth = depth

	plan, err := mapplanner.Plan(world, consts.MapTileWidth, consts.MapTileHeight, stageSeed, builderType)
	if err != nil {
		return zero, noEntity, err
	}
	level, err := mapspawner.Spawn(world, plan)
	if err != nil {
		return zero, noEntity, err
	}
	// フィールド寸法をこの階のステージメタへ記録する。生成物と同じ明示 key に束縛するため、
	// SwapTo が CurrentStage を最後に更新する順序に依存しない
	query.EnsureStageMeta(world, key).Level = level

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
	defName := query.GetDungeon(world).DefinitionName
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
		def, found := dungeon.GetDungeon(query.GetDungeon(world).DefinitionName)
		if !found {
			return fmt.Errorf("ダンジョン定義が見つかりません: %s", query.GetDungeon(world).DefinitionName)
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
	d := query.GetDungeon(world)
	// オーバーワールドへ戻ったら遺跡定義名をクリアする。残すと OnStart の再構築や
	// タイトルエフェクトが古い遺跡名を参照しうる。SwapTo 後は現ステージ=target なので現在地で判定する
	if query.IsOnOverworld(world) {
		d.DefinitionName = ""
		// 各ステージが自分の Level を StageMeta として保持するため、地上のメタが resume で
		// そのまま戻り、帯寸法を手で復元する必要はない。以前はグローバルな Level 1枚が遺跡進入で
		// 置き換わり、地上帰還で真っ暗・No Data・隊員配置失敗を招いていた。視界だけ強制再計算する
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
		def, found := dungeon.GetDungeon(defName)
		if !found {
			return fmt.Errorf("遺跡定義が見つかりません: %s", defName)
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
	d := query.GetDungeon(world)
	d.DefinitionName = defName

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

// OnStop はステートが停止される際に呼ばれる。
//
// 共存方式ではオーバーワールドと遺跡が同一 world に共存し、退避中ステージも保持される。
// かつてここで全フィールドエンティティを一括 purge していたが、これは「潜行を丸ごと捨てる
// 完全離脱」という旧概念の名残で、共存方式では退避ステージまで消しかねず有害。よって廃止する。
// world を捨てるのはタイトルへ戻る・ロードのときで、そこは MainMenuState.OnStart の全 entity
// 削除と save の ECS.Reset が担う。ステージ単位の破棄が要る場合は stage.Purge を明示的に呼ぶ。
func (st *DungeonState) OnStop(_ w.World) error { return nil }

// Update はゲームステートの更新処理を行う
func (st *DungeonState) Update(world w.World) (es.Transition[w.World], error) {
	// 全ダンジョン踏破の判定。街がオーバーワールドの地物になったため、旧・街帰還時でなく
	// オーバーワールド滞在時に判定する。判定条件は帯シフトと同じ「session保持かつ現ステージ深度0」。
	// SetEventActive は冪等で視聴後は再発火しないので、毎フレーム呼んでも一度だけ発火する
	if st.session != nil && query.IsOnOverworld(world) {
		gp := query.GetGameProgress(world)
		if gp.IsAllCleared(dungeon.GetAllDungeonNames()) {
			gp.SetEventActive(gc.EventAllCleared)
		}
	}

	// 全クリアイベントの表示
	if query.GetGameProgress(world).IsEventUnseen(gc.EventAllCleared) {
		query.GetGameProgress(world).MarkEventSeen(gc.EventAllCleared)
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			NewAllClearEventState,
		}}, nil
	}

	// キー入力をActionに変換
	if action, ok := st.HandleInput(world.Config); ok {
		if transition, err := st.DoAction(world, action); err != nil {
			return es.Transition[w.World]{}, err
		} else if transition.Type != es.TransNone {
			return transition, nil
		}
	}

	for _, updater := range []w.Updater{
		&gs.AnimationSystem{},
		&gs.DeadCleanupSystem{},
		&gs.TurnSystem{},
		&gs.VisionSystem{},
		&gs.CameraSystem{},
		&gs.HUDRenderingSystem{},
		&gs.StatsChangedSystem{},
		&gs.WeightDirtySystem{},
		&gs.VisualEffectSystem{},
	} {
		if sys, ok := world.Updaters[updater.String()]; ok {
			if err := sys.Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}

	// プレイヤー死亡チェック
	if st.checkPlayerDeath(world) {
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewGameOverMessageState}}, nil
	}

	// ステート遷移リクエストを処理
	transition, err := st.handleStateChangeRequest(world)
	if err != nil {
		return es.Transition[w.World]{}, err
	}
	if transition.Type != es.TransNone {
		return transition, nil
	}

	// BaseStateの共通処理を使用
	transition = st.ConsumeTransition()
	// 現ステージがオーバーワールドのときだけ前線を進め帯をシフトする。帯セッションは
	// オーバーワールド State だけが持ち、現ステージ深度0がオーバーワールドを表す。遺跡へ入ると
	// 同一 State 内で現ステージ深度が1以上へ変わり、そのあいだ帯を触らない。通常ダンジョンは
	// session が nil で除外される。死亡やリクエスト遷移で早期 return したフレームも触らない
	if st.session != nil && query.IsOnOverworld(world) && transition.Type == es.TransNone {
		st.session.UpdateFront(world)
		shifted, serr := st.session.MaybeShift(world)
		if serr != nil {
			return es.Transition[w.World]{}, serr
		}
		if shifted {
			// リベースでプレイヤーが中央へ動くが、カメラは Update 内で既に旧位置に合わせた後。
			// カメラを再センタリングしないと、シフトしたフレームで視点がジャンプしてチラつく
			if err := (&gs.CameraSystem{}).Update(world); err != nil {
				return es.Transition[w.World]{}, err
			}
		}
	}
	return transition, nil
}

// Draw はゲームステートの描画処理を行う
func (st *DungeonState) Draw(world w.World, screen *ebiten.Image) error {
	if st.baseImage != nil {
		screen.DrawImage(st.baseImage, nil)
	}

	for _, renderer := range []w.Renderer{
		&gs.RenderSpriteSystem{},
		&gs.FrostRenderSystem{},
		&gs.HUDRenderingSystem{},
		&gs.VisualEffectSystem{},
	} {
		if sys, ok := world.Renderers[renderer.String()]; ok {
			if err := sys.Draw(world, screen); err != nil {
				return err
			}
		}
	}

	return nil
}

// ================

// HandleInput はキー入力をActionに変換する
func (st *DungeonState) HandleInput(cfg *config.Config) (inputmapper.ActionID, bool) {
	keyboardInput := input.GetSharedKeyboardInput()

	if cfg.Debug && keyboardInput.IsKeyJustPressed(ebiten.KeySlash) {
		return inputmapper.ActionOpenDebugMenu, true
	}

	// ダンジョンメニュー
	if keyboardInput.IsKeyJustPressed(ebiten.KeyM) {
		return inputmapper.ActionOpenDungeonMenu, true
	}

	// インタラクションメニュー
	if keyboardInput.IsKeyJustPressed(ebiten.KeySpace) {
		return inputmapper.ActionOpenInteractionMenu, true
	}

	// 視界情報表示
	if keyboardInput.IsKeyJustPressed(ebiten.KeyX) {
		return inputmapper.ActionOpenFieldInfo, true
	}

	// 射撃モード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyF) {
		return inputmapper.ActionShoot, true
	}

	// 拾うモード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyG) {
		return inputmapper.ActionPickup, true
	}

	// 置くモード
	if keyboardInput.IsKeyJustPressed(ebiten.KeyP) {
		return inputmapper.ActionPlace, true
	}

	// 移動入力
	if action, ok := handleMoveInput(keyboardInput); ok {
		return action, true
	}

	// 待機キー（キーリピート対応）
	if keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyPeriod) {
		return inputmapper.ActionWait, true
	}

	// 相互作用キー（Enter）
	if keyboardInput.IsEnterJustPressedOnce() {
		return inputmapper.ActionInteract, true
	}

	// 武器スロット切り替え（1-5キー）
	if keyboardInput.IsKeyJustPressed(ebiten.Key1) {
		return inputmapper.ActionSwitchWeaponSlot1, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key2) {
		return inputmapper.ActionSwitchWeaponSlot2, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key3) {
		return inputmapper.ActionSwitchWeaponSlot3, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key4) {
		return inputmapper.ActionSwitchWeaponSlot4, true
	}
	if keyboardInput.IsKeyJustPressed(ebiten.Key5) {
		return inputmapper.ActionSwitchWeaponSlot5, true
	}

	return "", false
}

// DoAction はActionを実行する
//
//nolint:gocyclo // 多くのアクションを処理するためswitch文が大きくなる
func (st *DungeonState) DoAction(world w.World, action inputmapper.ActionID) (es.Transition[w.World], error) {
	// UI系アクションは常に実行可能
	switch action {
	case inputmapper.ActionOpenDungeonMenu, inputmapper.ActionOpenDebugMenu, inputmapper.ActionOpenInventory, inputmapper.ActionOpenInteractionMenu, inputmapper.ActionOpenFieldInfo, inputmapper.ActionShoot, inputmapper.ActionPickup, inputmapper.ActionPlace:
		// UI系はターンチェック不要
	default:
		// ゲーム内アクション（移動、攻撃など）はターンチェックが必要
		if !query.CanPlayerAct(world) {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
		// プレイヤーが継続アクション中は新しいアクションを受け付けない
		if playerEntity, err := query.GetPlayerEntity(world); err == nil && query.HasActivity(world, playerEntity) {
			return es.Transition[w.World]{Type: es.TransNone}, nil
		}
	}

	switch action {
	// UI系アクション（ステート遷移）
	case inputmapper.ActionOpenDungeonMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDungeonMenuState}}, nil
	case inputmapper.ActionOpenDebugMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewDebugMenuState}}, nil
	case inputmapper.ActionOpenInventory:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{NewInventoryMenuState}}, nil
	case inputmapper.ActionOpenInteractionMenu:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewInteractionMenuState(world) },
		}}, nil
	case inputmapper.ActionOpenFieldInfo:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &LookAroundState{}, nil },
		}}, nil
	case inputmapper.ActionShoot:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &ShootingState{}, nil },
		}}, nil
	case inputmapper.ActionPickup:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &PickupState{}, nil },
		}}, nil
	case inputmapper.ActionPlace:
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return &PlaceState{}, nil },
		}}, nil

	// 移動系アクション
	case inputmapper.ActionMoveNorth:
		if err := activity.ExecuteMoveAction(world, gc.DirectionUp); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouth:
		if err := activity.ExecuteMoveAction(world, gc.DirectionDown); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveEast:
		if err := activity.ExecuteMoveAction(world, gc.DirectionRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveWest:
		if err := activity.ExecuteMoveAction(world, gc.DirectionLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveNorthEast:
		if err := activity.ExecuteMoveAction(world, gc.DirectionUpRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveNorthWest:
		if err := activity.ExecuteMoveAction(world, gc.DirectionUpLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouthEast:
		if err := activity.ExecuteMoveAction(world, gc.DirectionDownRight); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionMoveSouthWest:
		if err := activity.ExecuteMoveAction(world, gc.DirectionDownLeft); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionWait:
		if err := activity.ExecuteWaitAction(world); err != nil {
			return es.Transition[w.World]{Type: es.TransNone}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil

	// 相互作用系アクション
	case inputmapper.ActionInteract:
		actions := GetSameTileManualActions(world)
		switch len(actions) {
		case 0:
			// 何もしない
		case 1:
			playerEntity, err := query.GetPlayerEntity(world)
			if err != nil {
				return es.Transition[w.World]{Type: es.TransNone}, err
			}
			if _, err := activity.ExecuteInteraction(playerEntity, actions[0].Target, actions[0].Interaction, world); err != nil {
				return es.Transition[w.World]{Type: es.TransNone}, err
			}
		default:
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return newActionChoiceMenu(actions), nil },
			}}, nil
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil

	// 武器スロット切り替え系アクション
	case inputmapper.ActionSwitchWeaponSlot1:
		st.switchWeaponSlot(world, 1)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot2:
		st.switchWeaponSlot(world, 2)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot3:
		st.switchWeaponSlot(world, 3)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot4:
		st.switchWeaponSlot(world, 4)
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case inputmapper.ActionSwitchWeaponSlot5:
		st.switchWeaponSlot(world, 5)
		return es.Transition[w.World]{Type: es.TransNone}, nil

	default:
		return es.Transition[w.World]{}, fmt.Errorf("未知のアクション: %s", action)
	}
}

// ================

// checkPlayerDeath はプレイヤーの死亡状態をチェックする
func (st *DungeonState) checkPlayerDeath(world w.World) bool {
	playerDead := false
	playerDeadQuery := ecs.NewFilter2[gc.Player, gc.Dead](world.ECS).Query()
	for playerDeadQuery.Next() {
		playerDead = true
	}
	return playerDead
}

// handleStateChangeRequest はステート遷移リクエストを消費し、対応する遷移を返す
func (st *DungeonState) handleStateChangeRequest(world w.World) (es.Transition[w.World], error) {
	req := lifecycle.ConsumeStateChange(world)
	if req == nil {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	switch p := req.Payload.(type) {
	case gc.ShowDialog:
		// SpeakerEntityからNameを取得
		if !world.Components.Name.Has(p.SpeakerEntity) {
			return es.Transition[w.World]{}, fmt.Errorf("speaker entity does not have Name component")
		}
		nameComp := world.Components.Name.Get(p.SpeakerEntity)
		speakerName := nameComp.Name

		// NPCの種類に応じて専用ステートを返す
		switch p.MessageKey {
		case "merchant_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMerchantDialogState(speakerName) },
			}}, nil
		case "doctor_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewDoctorDialogState(speakerName) },
			}}, nil
		case "tavern_keeper_greeting":
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewTavernKeeperDialogState(speakerName) },
			}}, nil
		default:
			// 通常の会話はdialoguesから取得
			dialogMessage := messagedata.GetDialogue(p.MessageKey, speakerName)
			return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
				func() (es.State[w.World], error) { return NewMessageState(dialogMessage) },
			}}, nil
		}
	case gc.WarpDescend:
		// 共存方式の下り。同一 State 内で swapTo する。現階は退避され再訪で復元できる
		if err := st.descend(world); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case gc.WarpAscend:
		// 上り階段に結線があればそこへ移動する。浅い階でも遺跡→地上でも同一機構。
		// 全ダンジョンはオーバーワールド入口から入り、生成時に戻り先が結線される。よって
		// handled=false は結線焼き込みの取りこぼし＝バグであり、黙って握り潰さず error で落とす
		handled, err := st.ascend(world)
		if err != nil {
			return es.Transition[w.World]{}, err
		}
		if !handled {
			return es.Transition[w.World]{}, fmt.Errorf("最上階の上り階段に戻り先の結線がありません")
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case gc.WarpDungeonEnter:
		// オーバーワールドから遺跡へ入る。同一 State 内 swapTo で帯を退避し遺跡へ切り替える
		if err := st.enterDungeon(world, p.DefinitionName); err != nil {
			return es.Transition[w.World]{}, err
		}
		return es.Transition[w.World]{Type: es.TransNone}, nil
	case gc.OpenStorage:
		// 収納メニューを開く
		return es.Transition[w.World]{Type: es.TransPush, NewStateFuncs: []es.StateFactory[w.World]{
			func() (es.State[w.World], error) { return NewStorageMenuState(p.StorageEntity) },
		}}, nil
	default:
		// この switch で扱わない種別。未実装の scaffold もここに落ちる
		return es.Transition[w.World]{}, fmt.Errorf("未処理のStateChangeRequest: %T", req.Payload)
	}
}

// switchWeaponSlot は指定されたスロット番号（1-5）に武器を切り替える
func (st *DungeonState) switchWeaponSlot(world w.World, slotNumber int) {
	query.GetWeaponSelection(world).Slot = slotNumber

	// プレイヤーの武器スロット情報を取得してログメッセージを出力
	query.Player(world, func(playerEntity ecs.Entity) {
		weapons := query.GetWeapons(world, playerEntity)
		weaponIndex := slotNumber - 1 // 1-based to 0-based
		weapon := weapons[weaponIndex]

		if weapon != nil {
			// 武器が装備されている場合は武器名を表示
			if nameComp := world.Components.Name.Get(*weapon); nameComp != nil {
				weaponName := nameComp.Name
				gamelog.New(query.GetGameLog(world)).
					ItemName(weaponName).
					Append("を構えた").
					Log()
			}
		}
	})
}

// handleMoveInput は8方向移動のキー入力を処理する
func handleMoveInput(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	// Shift押下中は斜め移動モード。2キー同時押しの斜め移動のみ受け付ける。
	// IsKeyPressedWithRepeatは副作用があるため、Shift判定を先に行い不要な呼び出しを避ける
	if keyboardInput.IsKeyPressed(ebiten.KeyShift) {
		return handleShiftDiagonalInput(keyboardInput)
	}

	upPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyW) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyS) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyA) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyLeft)
	rightPressed := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyD) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyRight)

	if upPressed {
		return inputmapper.ActionMoveNorth, true
	}
	if downPressed {
		return inputmapper.ActionMoveSouth, true
	}
	if leftPressed {
		return inputmapper.ActionMoveWest, true
	}
	if rightPressed {
		return inputmapper.ActionMoveEast, true
	}

	return "", false
}

// handleShiftDiagonalInput はShift押下中の斜め移動入力を処理する。
// 縦軸のIsKeyPressedWithRepeatのみをリピートタイミングの制御に使い、横軸はIsKeyPressedで判定する。
// 両軸のリピートをOR条件にするとリピート頻度が2倍になるため、片軸のみをドライバーにする
func handleShiftDiagonalInput(keyboardInput input.KeyboardInput) (inputmapper.ActionID, bool) {
	upRepeat := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyW) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyUp)
	downRepeat := keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyS) || keyboardInput.IsKeyPressedWithRepeat(ebiten.KeyDown)
	leftHeld := keyboardInput.IsKeyPressed(ebiten.KeyA) || keyboardInput.IsKeyPressed(ebiten.KeyLeft)
	rightHeld := keyboardInput.IsKeyPressed(ebiten.KeyD) || keyboardInput.IsKeyPressed(ebiten.KeyRight)

	if upRepeat && leftHeld {
		return inputmapper.ActionMoveNorthWest, true
	}
	if upRepeat && rightHeld {
		return inputmapper.ActionMoveNorthEast, true
	}
	if downRepeat && leftHeld {
		return inputmapper.ActionMoveSouthWest, true
	}
	if downRepeat && rightHeld {
		return inputmapper.ActionMoveSouthEast, true
	}
	return "", false
}
