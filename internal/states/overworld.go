package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/systems"
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
func NewOverworldState(runSeed uint64, chunkW, chunkH consts.Tile, k worldstream.ChunkX, planner mapplanner.PlannerType) es.StateFactory[w.World] {
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
// 帯パラメータの seed・chunkW・chunkH・k・eastIndex は OnStart が Dungeon.SeamlessBand から
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

// OnStart は K チャンク分の初期帯を生成し、プレイヤーを中央チャンクへ置く。
func (st *OverworldState) OnStart(world w.World) error {
	sw := world.Resources.ScreenDimensions.Width
	sh := world.Resources.ScreenDimensions.Height
	if sw > 0 && sh > 0 {
		st.baseImage = ebiten.NewImage(sw, sh)
		st.baseImage.Fill(theme.ScreenBackground)
	}

	d := query.GetDungeon(world)
	sb := &d.SeamlessBand

	// 視界の強制再計算を促す。VisionSystem は world.Updaters に居座る永続インスタンスで、
	// Depth/DefinitionName が変わらないと内部キャッシュを無効化しない。オーバーワールドは常に
	// Depth=0 なのでフロア変化が起きず、ロード復元では serde が空にした VisibleTiles が
	// stale な isInitialized のまま再計算されず真っ暗になる。ここで一度だけ強制する。
	d.NeedsForceUpdate = true

	// ロード復元: 帯タイル・Level・プレイヤーは serde で復元済み。
	// ここでは Band ドライバと ChunkGen を永続状態から再構築するだけでよい。再生成はしない。
	if sb.Active {
		st.runSeed, st.chunkW, st.chunkH = sb.RunSeed, sb.ChunkW, sb.ChunkH
		st.band = worldstream.NewBandAt(sb.ChunkW, worldstream.ChunkX(sb.K), worldstream.ChunkX(sb.EastIndex))
		st.gen = overworld.NewChunkGen(world, sb.RunSeed, sb.ChunkW, sb.ChunkH, st.planner)
		query.InvalidateSpatialIndex(world)
		return nil
	}

	// 新規開始: 帯状態を Dungeon に記録してセーブに対応し、初期帯を生成してプレイヤーを配置する
	sb.Active = true
	sb.RunSeed = st.runSeed
	sb.EastIndex = int(st.band.EastIndex())
	sb.ChunkW = st.chunkW
	sb.ChunkH = st.chunkH
	sb.K = int(st.band.K())

	// 初期帯 ＝ K*chunkW × chunkH の単一マップを決定的生成する
	d.ExploredTiles = make(map[gc.GridElement]bool)
	st.gen = overworld.NewChunkGen(world, st.runSeed, st.chunkW, st.chunkH, st.planner)
	if err := st.generateBandChunks(world); err != nil {
		return err
	}

	// プレイヤーを中央チャンクの中央へ。居なければ生成、居れば移動
	cx := int((st.band.K() / 2).Tiles(st.chunkW) + st.chunkW/2)
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

// syncBandState は Band の現在 eastIndex を Dungeon の永続状態へ書き戻す。これでセーブに反映される。
func (st *OverworldState) syncBandState(world w.World) {
	query.GetDungeon(world).SeamlessBand.EastIndex = int(st.band.EastIndex())
}

// generateBandChunks は Level を帯全幅に設定し、K チャンクを各スロットへ決定的生成する。
// OnStart の新規開始から呼ばれる。Level 設定は帯幅が不変なので再設定しても冪等で無害。
func (st *OverworldState) generateBandChunks(world w.World) error {
	query.GetDungeon(world).Level = gc.Level{TileWidth: st.band.Width(), TileHeight: st.chunkH}
	for i := range st.band.K() {
		if err := st.gen(i, i.Tiles(st.chunkW)); err != nil {
			return fmt.Errorf("チャンク生成失敗 (slot=%d): %w", i, err)
		}
	}
	return nil
}

// OnPause/OnResume は DungeonState の no-op を継承し、オーバーライドしない。
//
// 射撃・観察・ダンジョンメニュー等のオーバーレイは TransPush で載るため OnPause/OnResume が
// 呼ばれるが、これらは同じ世界を描画・操作するだけなので帯タイルはそのまま残す必要がある。
// ここで帯を退避/再生成すると、オーバーレイ進入で帯タイルが消えて画面が黒くなり、
// 復帰時の MovePlayerToPosition が隊員を再配置してしまう。
//
// 将来オーバーワールドにポータルを足して遺跡へ入れるようにする場合、帯の退避は汎用フックの
// OnPause ではなく「遺跡進入」専用の経路で行う。汎用フックはオーバーレイと区別できないため。
// 設計 docs/design/20260717_60.md §4。

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

// maybeShift はプレイヤーが中央チャンクを出ていれば帯をシフトする。§2.5 のターン境界フック。
//
// 座標を平行移動する破壊的操作なので、ターンが完全に解決した安定点でのみ行う。すなわち
// プレイヤーターンの Player フェーズかつプレイヤーが継続アクティビティ中でないとき。
// これによりアニメ補間中・移動 Activity 実行中のシフトを避ける。
func (st *OverworldState) maybeShift(world w.World) error {
	if query.GetTurnState(world).Phase != gc.TurnPhasePlayer {
		return nil
	}
	// Update は死亡チェック後にのみ maybeShift へ到達するため、ここでプレイヤーは存在するはず。
	// 不在は異常なので伝播する。cullDistantSolo と同じ方針
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return fmt.Errorf("シフト判定にプレイヤーが必要: %w", err)
	}
	if query.HasActivity(world, playerEntity) {
		return nil
	}
	// 中央チャンクに収まるまでシフトを繰り返す。設計 §2.1 の while 相当。
	// 各シフトはプレイヤーを chunkW ぶん中央へ寄せるため、必ず有限回で収束する。
	shifted := false
	for {
		localX := world.Components.GridElement.Get(playerEntity).X
		if st.band.ShouldShiftEast(localX) {
			if err := st.band.ShiftEast(world, st.gen); err != nil {
				return err
			}
			st.syncBandState(world)
			shifted = true
			continue
		}
		// 西シフトは寄り道からの復帰時のみ。ラン開始の eastIndex=0 より西には
		// 何も生成されていないため、eastIndex を負にする西シフトは行わない。
		// プレイヤーは帯西端の localX=0 の境界で自然に止まる
		if st.band.ShouldShiftWest(localX) && st.band.EastIndex() > 0 {
			if err := st.band.ShiftWest(world, st.gen); err != nil {
				return err
			}
			st.syncBandState(world)
			shifted = true
			continue
		}
		break
	}

	if shifted {
		// リベースでプレイヤーが中央へ動くが、カメラは Update 内で既に旧位置に合わせた後。
		// カメラを再センタリングしないと、シフトしたフレームで視点がジャンプしてチラつく
		if err := (&gs.CameraSystem{}).Update(world); err != nil {
			return err
		}
	}
	return nil
}
