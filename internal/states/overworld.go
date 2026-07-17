package states

import (
	"fmt"
	"maps"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// OverworldState はシームレスワールドを東へ延々と歩く探索ステート。
//
// DungeonState を埋め込み、入力・システム列・描画・遷移処理をそのまま再利用する。
// OverworldState 固有なのは「フロアを作り直さず、アクティブ帯をスライドさせ続ける」点だけで、
// OnStart で初期帯を生成し、Update のターン境界でシフト判定する。詳細設計は
// docs/design/20260717_60.md §6。
type OverworldState struct {
	*DungeonState
	band    *worldstream.Band
	gen     worldstream.ChunkGen
	runSeed uint64
	chunkW  consts.Tile
	chunkH  consts.Tile
	planner mapplanner.PlannerType

	// 遺跡へ TransPush する際に退避する動的状態（決定的に再生成できないもの）。
	// 帯タイルは seed 決定的なので保存せず OnResume で再生成する。
	savedPlayerPos *gc.GridElement
	savedExplored  map[gc.GridElement]bool
}

var _ es.State[w.World] = &OverworldState{}
var _ es.ActionHandler[w.World] = &OverworldState{}

// String はステート名を返す
func (st *OverworldState) String() string { return "Overworld" }

// NewOverworldState はシームレスワールドステートのファクトリを返す。
// chunkW×chunkH のチャンクを k 枚並べた帯を runSeed から決定的生成する。
func NewOverworldState(runSeed uint64, chunkW, chunkH consts.Tile, k int, planner mapplanner.PlannerType) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &OverworldState{
			DungeonState: &DungeonState{},
			band:         worldstream.NewBand(chunkW, k),
			runSeed:      runSeed,
			chunkW:       chunkW,
			chunkH:       chunkH,
			planner:      planner,
		}, nil
	}
}

// NewOverworldStateForLoad はセーブから復元する際のファクトリを返す。
// 帯パラメータ（seed/chunkW/chunkH/k/eastIndex）は OnStart が Dungeon.SeamlessBand から
// 読み取って再構築するため、ここでは planner だけ指定すればよい。
func NewOverworldStateForLoad(planner mapplanner.PlannerType) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &OverworldState{
			DungeonState: &DungeonState{},
			band:         worldstream.NewBand(1, 1), // OnStart で SeamlessBand から再構築される
			planner:      planner,
		}, nil
	}
}

// OnStart は初期帯（K チャンク）を生成し、プレイヤーを中央チャンクへ置く。
func (st *OverworldState) OnStart(world w.World) error {
	sw := world.Resources.ScreenDimensions.Width
	sh := world.Resources.ScreenDimensions.Height
	if sw > 0 && sh > 0 {
		st.baseImage = ebiten.NewImage(sw, sh)
		st.baseImage.Fill(theme.ScreenBackground)
	}

	d := query.GetDungeon(world)
	sb := &d.SeamlessBand

	// ロード復元: 帯タイル・Level・プレイヤーは serde で復元済み。
	// ここでは Band ドライバと ChunkGen を永続状態から再構築するだけでよい（再生成しない）。
	if sb.Active {
		st.runSeed, st.chunkW, st.chunkH = sb.RunSeed, sb.ChunkW, sb.ChunkH
		st.band = worldstream.NewBandAt(sb.ChunkW, sb.K, sb.EastIndex)
		st.gen = overworld.NewChunkGen(world, sb.RunSeed, sb.ChunkW, sb.ChunkH, st.planner)
		query.InvalidateSpatialIndex(world)
		return nil
	}

	// 新規開始: 帯状態を Dungeon に記録し（セーブ対応）、初期帯を生成してプレイヤーを配置する
	sb.Active = true
	sb.RunSeed = st.runSeed
	sb.EastIndex = st.band.EastIndex()
	sb.ChunkW = st.chunkW
	sb.ChunkH = st.chunkH
	sb.K = st.band.K()

	// 初期帯 ＝ K*chunkW × chunkH の単一マップを決定的生成する
	d.ExploredTiles = make(map[gc.GridElement]bool)
	st.gen = overworld.NewChunkGen(world, st.runSeed, st.chunkW, st.chunkH, st.planner)
	if err := st.generateBandChunks(world); err != nil {
		return err
	}

	// プレイヤーを中央チャンクの中央へ。居なければ生成、居れば移動
	cx := int(consts.Tile(st.band.K()/2)*st.chunkW + st.chunkW/2)
	cy := int(st.chunkH / 2)
	if _, err := query.GetPlayerEntity(world); err != nil {
		if _, serr := lifecycle.SpawnPlayer(world, cx, cy, "Ash"); serr != nil {
			return fmt.Errorf("プレイヤー生成失敗: %w", serr)
		}
	} else if merr := lifecycle.MovePlayerToPosition(world, cx, cy); merr != nil {
		return fmt.Errorf("プレイヤー配置失敗: %w", merr)
	}

	query.InvalidateSpatialIndex(world)
	return nil
}

// syncBandState は Band の現在 eastIndex を Dungeon の永続状態へ書き戻す（セーブに反映させる）。
func (st *OverworldState) syncBandState(world w.World) {
	query.GetDungeon(world).SeamlessBand.EastIndex = st.band.EastIndex()
}

// generateBandChunks は Level を帯全幅に設定し、K チャンクを各スロットへ決定的生成する。
func (st *OverworldState) generateBandChunks(world w.World) error {
	query.GetDungeon(world).Level = gc.Level{TileWidth: st.band.Width(), TileHeight: st.chunkH}
	for i := range st.band.K() {
		if err := st.gen(i, consts.Tile(i)*st.chunkW); err != nil {
			return fmt.Errorf("チャンク生成失敗 (slot=%d): %w", i, err)
		}
	}
	return nil
}

// OnPause は遺跡へ TransPush する直前に呼ばれる。帯を退避する。
//
// 帯タイルは seed 決定的なので保存せず削除し、遺跡がクリーンな座標空間で生成できるようにする
// （プレイヤー・隊員は残す）。決定的に戻せない動的状態（プレイヤー位置・探索済み）だけ保存する。
func (st *OverworldState) OnPause(world w.World) error {
	if p, err := query.GetPlayerEntity(world); err == nil {
		pos := *world.Components.GridElement.Get(p)
		st.savedPlayerPos = &pos
	}
	// 参照でなくコピーを退避する。遺跡側が ExploredTiles を再代入せず in-place 変異しても汚染されない
	st.savedExplored = maps.Clone(query.GetDungeon(world).ExploredTiles)
	// 帯タイルを消す（プレイヤー・隊員は残す）。遺跡タイルとの重なりを防ぐ
	worldstream.RemoveEntitiesInXRange(world, 0, st.band.Width(), worldstream.KeepPlayerAndSquad(world))
	return nil
}

// OnResume は遺跡から TransPop で戻った際に呼ばれる。帯を再構築する。
//
// 遺跡の OnStop が非プレイヤーエンティティを全削除しているため、帯を決定的に再生成し、
// 退避したプレイヤー位置・探索済みを復元する。
func (st *OverworldState) OnResume(world w.World) error {
	// 再生成前に帯領域を掃除して自己完結させる（遺跡側 OnStop の副作用に依存せず、
	// 二重生成を防ぐ）。プレイヤー・隊員は残す
	worldstream.RemoveEntitiesInXRange(world, 0, st.band.Width(), worldstream.KeepPlayerAndSquad(world))
	if err := st.generateBandChunks(world); err != nil {
		return err
	}
	if st.savedExplored != nil {
		query.GetDungeon(world).ExploredTiles = st.savedExplored
		st.savedExplored = nil
	}
	if st.savedPlayerPos != nil {
		if err := lifecycle.MovePlayerToPosition(world, int(st.savedPlayerPos.X), int(st.savedPlayerPos.Y)); err != nil {
			return fmt.Errorf("プレイヤー位置の復元に失敗: %w", err)
		}
		st.savedPlayerPos = nil
	}
	query.InvalidateSpatialIndex(world)
	return nil
}

// Update は DungeonState の共通処理を実行後、ターン境界で帯をシフトする。
func (st *OverworldState) Update(world w.World) (es.Transition[w.World], error) {
	trans, err := st.DungeonState.Update(world)
	if err != nil || trans.Type != es.TransNone {
		return trans, err
	}
	if serr := st.maybeShift(world); serr != nil {
		return es.Transition[w.World]{}, serr
	}
	return trans, nil
}

// maybeShift はプレイヤーが中央チャンクを出ていれば帯をシフトする（§2.5 のターン境界フック）。
//
// 座標を平行移動する破壊的操作なので、ターンが完全に解決した安定点でのみ行う:
// プレイヤーターン（Player フェーズ）かつプレイヤーが継続アクティビティ中でないとき。
// これによりアニメ補間中・移動 Activity 実行中のシフトを避ける。
func (st *OverworldState) maybeShift(world w.World) error {
	if query.GetTurnState(world).Phase != gc.TurnPhasePlayer {
		return nil
	}
	// Update は死亡チェック後にのみ maybeShift へ到達するため、ここでプレイヤーは存在するはず。
	// 不在は異常なので伝播する（cullDistantSolo と同じ方針）
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return fmt.Errorf("シフト判定にプレイヤーが必要: %w", err)
	}
	if query.HasActivity(world, playerEntity) {
		return nil
	}
	// 中央チャンクに収まるまでシフトを繰り返す（設計 §2.1 の while 相当）。
	// 各シフトはプレイヤーを chunkW ぶん中央へ寄せるため、必ず有限回で収束する。
	for {
		localX := world.Components.GridElement.Get(playerEntity).X
		switch {
		case st.band.ShouldShiftEast(localX):
			if err := st.band.ShiftEast(world, st.gen); err != nil {
				return err
			}
			st.syncBandState(world)
		case st.band.ShouldShiftWest(localX) && st.band.EastIndex() > 0:
			// 西シフトは寄り道からの復帰時のみ。ラン開始（eastIndex=0）より西には
			// 何も生成されていないため、eastIndex を負にする西シフトは行わない。
			// プレイヤーは帯西端（localX=0 の境界）で自然に止まる
			if err := st.band.ShiftWest(world, st.gen); err != nil {
				return err
			}
			st.syncBandState(world)
		default:
			return nil
		}
	}
}
