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
// OverworldState 固有なのは「フロアを作り直さず、アクティブ帯をスライドさせ続ける」点だけ。
// OnStart で初期帯を生成し、Update のターン境界でシフト判定する。
type OverworldState struct {
	*DungeonState
	planner mapplanner.PlannerType
	newGame *NewGameParams // 新規開始の帯パラメータ。ロード復元では nil

	// 以下は OnStart で確定する実行時状態
	band     *worldstream.Band
	gen      worldstream.ChunkGen
	frontCfg worldstream.FrontConfig
}

var _ es.State[w.World] = &OverworldState{}
var _ es.ActionHandler[w.World] = &OverworldState{}

// String はステート名を返す
func (st *OverworldState) String() string { return "Overworld" }

// NewGameParams は新規オーバーワールド開始時の帯生成パラメータ。
// chunkW×chunkH のチャンクを K 枚並べた帯を RunSeed から決定的生成する。
type NewGameParams struct {
	RunSeed uint64
	ChunkW  consts.Tile
	ChunkH  consts.Tile
	K       consts.ChunkX
}

// NewOverworldState はシームレスワールドステートのファクトリを返す。
//
// params が非 nil なら新規開始として初期帯を生成する。nil ならセーブからの復元とみなし、
// 帯パラメータは OnStart が Dungeon.SeamlessBand から読み取って再構築する。
// 新規開始とロード復元の初期化本体は startNewBand・restoreFromSave に分かれている。
func NewOverworldState(planner mapplanner.PlannerType, params *NewGameParams) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &OverworldState{
			DungeonState: &DungeonState{},
			planner:      planner,
			newGame:      params,
		}, nil
	}
}

// OnStart は帯ドライバを用意する。新規開始なら初期帯を生成し、ロード復元なら
// セーブ済みの SeamlessBand から Band と ChunkGen を作り直す。分岐の本体は
// startNewBand・restoreFromSave に分けてあり、OnStart 自身はどちらを呼ぶかだけを決める。
func (st *OverworldState) OnStart(world w.World) error {
	sw := world.Resources.ScreenDimensions.Width
	sh := world.Resources.ScreenDimensions.Height
	if sw > 0 && sh > 0 {
		st.baseImage = ebiten.NewImage(sw, sh)
		st.baseImage.Fill(theme.ScreenBackground)
	}

	d := query.GetDungeon(world)

	// 視界の強制再計算を促す。VisionSystem は world.Updaters に居座る永続インスタンスで、
	// Depth/DefinitionName が変わらないと内部キャッシュを無効化しない。オーバーワールドは常に
	// Depth=0 なのでフロア変化が起きず、ロード復元では serde が空にした VisibleTiles が
	// stale な isInitialized のまま再計算されず真っ暗になる。ここで一度だけ強制する。
	d.NeedsForceUpdate = true

	sb := &d.SeamlessBand
	if sb.Active {
		return st.restoreFromSave(world, sb)
	}
	return st.startNewBand(world, sb)
}

// restoreFromSave はセーブ済みの SeamlessBand から Band ドライバと ChunkGen を再構築する。
// 帯タイル・Level・プレイヤーは serde で復元済みなので再生成はしない。
func (st *OverworldState) restoreFromSave(world w.World, sb *gc.SeamlessBand) error {
	st.band = worldstream.NewBandAt(sb.ChunkW, sb.K, sb.EastIndex)
	st.gen = overworld.NewChunkGen(world, sb.RunSeed, sb.ChunkW, sb.ChunkH, st.planner)
	st.frontCfg = frontCfgFromBand(sb)
	query.InvalidateSpatialIndex(world)
	return nil
}

// frontCfgFromBand は永続状態から寒波前線の前進パラメータを復元する。
func frontCfgFromBand(sb *gc.SeamlessBand) worldstream.FrontConfig {
	return worldstream.FrontConfig{
		StartEast:    worldstream.AbsTileX(sb.FrontStartAbsX),
		ColdWidth:    sb.FrontColdWidth,
		AdvanceTurns: sb.FrontAdvanceTurns,
		Step:         sb.FrontStep,
	}
}

// front は総経過ターン数から寒波前線の現在位置を導出する。
func (st *OverworldState) front(world w.World) worldstream.Front {
	totalTurns := query.GetDungeon(world).GameTime.TotalTurns
	return worldstream.FrontAt(st.frontCfg, totalTurns)
}

// startNewBand は新規開始として初期帯を決定的生成し、帯状態を SeamlessBand へ記録し、
// プレイヤーを中央チャンクへ置く。帯パラメータは newGame から取る。nil なら誤用なので弾く。
func (st *OverworldState) startNewBand(world w.World, sb *gc.SeamlessBand) error {
	p := st.newGame
	if p == nil {
		return fmt.Errorf("新規オーバーワールドの開始には帯パラメータが必要")
	}
	st.band = worldstream.NewBand(p.ChunkW, p.K)
	st.gen = overworld.NewChunkGen(world, p.RunSeed, p.ChunkW, p.ChunkH, st.planner)

	// 帯状態を Dungeon に記録してセーブに対応する
	sb.Active = true
	sb.RunSeed = p.RunSeed
	sb.EastIndex = st.band.EastIndex()
	sb.ChunkW = p.ChunkW
	sb.ChunkH = p.ChunkH
	sb.K = st.band.K()

	// 寒波前線を初期化する。極低温ゾーン東端をプレイヤーの西へ置いて背後から迫らせる。
	// 速度と幅は暫定値で、凍結効果を入れる後続増分でバランス調整する。
	st.frontCfg = worldstream.FrontConfig{
		StartEast:    worldstream.BandOriginX(st.band.EastIndex(), p.ChunkW) - worldstream.AbsTileX(p.ChunkW),
		ColdWidth:    p.ChunkW * 2,
		AdvanceTurns: 3,
		Step:         1,
	}
	sb.FrontActive = true
	sb.FrontStartAbsX = consts.Tile(st.frontCfg.StartEast)
	sb.FrontColdWidth = st.frontCfg.ColdWidth
	sb.FrontAdvanceTurns = st.frontCfg.AdvanceTurns
	sb.FrontStep = st.frontCfg.Step

	// 初期帯 ＝ K*chunkW × chunkH の単一マップを決定的生成する
	query.GetDungeon(world).ExploredTiles = make(map[gc.GridElement]bool)
	if err := st.generateBandChunks(world, p.ChunkW, p.ChunkH); err != nil {
		return err
	}

	// プレイヤーを中央チャンクの中央へ。居なければ生成、居れば移動
	cx := int((st.band.K() / 2).Tiles(p.ChunkW) + p.ChunkW/2)
	cy := int(p.ChunkH / 2)
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
	query.GetDungeon(world).SeamlessBand.EastIndex = st.band.EastIndex()
}

// generateBandChunks は Level を帯全幅に設定し、K チャンクを各スロットへ決定的生成する。
// startNewBand から呼ばれる。Level 設定は帯幅が不変なので再設定しても冪等で無害。
func (st *OverworldState) generateBandChunks(world w.World, chunkW, chunkH consts.Tile) error {
	query.GetDungeon(world).Level = gc.Level{TileWidth: st.band.Width(), TileHeight: chunkH}
	for i := range st.band.K() {
		if err := st.gen(i, i.Tiles(chunkW)); err != nil {
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

// Update は DungeonState の共通処理を実行後、寒波前線を進め、ターン境界で帯をシフトする。
func (st *OverworldState) Update(world w.World) (es.Transition[w.World], error) {
	trans, err := st.DungeonState.Update(world)
	if err != nil || trans.Type != es.TransNone {
		return trans, err
	}
	st.updateFront(world)
	if serr := st.maybeShift(world); serr != nil {
		return es.Transition[w.World]{}, serr
	}
	return trans, nil
}

// updateFront は総ターン数から導出した寒波前線の現在位置を永続状態へ反映する。
// 位置は導出値なので毎フレーム書いても冪等。描画や凍結効果はこの FrontEastAbsX を読む。
func (st *OverworldState) updateFront(world w.World) {
	sb := &query.GetDungeon(world).SeamlessBand
	if !sb.FrontActive {
		return
	}
	sb.FrontEastAbsX = consts.Tile(st.front(world).East)
}

// maybeShift はプレイヤーが中央チャンクを出ていれば帯をシフトする。
//
// 座標を平行移動する破壊的操作なので、ターンが完全に解決した安定点でのみ行う。すなわち
// プレイヤーターンの Player フェーズかつプレイヤーが継続アクティビティ中でないとき。
// これによりアニメ補間中・移動 Activity 実行中のシフトを避ける。
func (st *OverworldState) maybeShift(world w.World) error {
	if query.GetTurnState(world).Phase != gc.TurnPhasePlayer {
		return nil
	}
	// Update は死亡チェック後にのみ maybeShift へ到達するため、ここでプレイヤーは存在するはず。
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return fmt.Errorf("シフト判定にプレイヤーが必要: %w", err)
	}
	if query.HasActivity(world, playerEntity) {
		return nil
	}
	// 中央チャンクに収まるまでシフトを繰り返す。
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
