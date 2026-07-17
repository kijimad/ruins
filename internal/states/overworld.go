package states

import (
	"fmt"

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

// OnStart は初期帯（K チャンク）を生成し、プレイヤーを中央チャンクへ置く。
func (st *OverworldState) OnStart(world w.World) error {
	sw := world.Resources.ScreenDimensions.Width
	sh := world.Resources.ScreenDimensions.Height
	if sw > 0 && sh > 0 {
		st.baseImage = ebiten.NewImage(sw, sh)
		st.baseImage.Fill(theme.ScreenBackground)
	}

	// 帯 ＝ K*chunkW × chunkH の単一マップ
	d := query.GetDungeon(world)
	d.Level = gc.Level{TileWidth: st.band.Width(), TileHeight: st.chunkH}
	d.ExploredTiles = make(map[gc.GridElement]bool)

	// 初期帯: K チャンクを各スロットへ決定的生成
	st.gen = overworld.NewChunkGen(world, st.runSeed, st.chunkW, st.chunkH, st.planner)
	for i := range st.band.K() {
		if err := st.gen(i, consts.Tile(i)*st.chunkW); err != nil {
			return fmt.Errorf("初期チャンク生成失敗 (slot=%d): %w", i, err)
		}
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
	localX := world.Components.GridElement.Get(playerEntity).X
	switch {
	case st.band.ShouldShiftEast(localX):
		return st.band.ShiftEast(world, st.gen)
	case st.band.ShouldShiftWest(localX):
		return st.band.ShiftWest(world, st.gen)
	}
	return nil
}
