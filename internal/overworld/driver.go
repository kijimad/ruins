package overworld

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
)

// NewGameParams は新規オーバーワールド開始時のプレイ固有パラメータ。
// プレイごとに変わるのは RunSeed だけ。帯形状は OverworldDefinition マスタが持つ。
type NewGameParams struct {
	RunSeed uint64
}

const (
	// frontAdvanceTurns は前線が frontStep タイル前進するのに要するターン数。大きいほどゆるやか。
	// 1500ターン/日なので 20 なら 75タイル/日。開始時に背後25タイルなら追いつくまで約500ターン≈0.33日
	frontAdvanceTurns consts.Turn = 20
	// frontStep は1回の前進量。タイル単位
	frontStep consts.Tile = 1
	// frontColdWidthChunks は極低温ゾーンの幅。チャンク数
	frontColdWidthChunks consts.Chunk = 2
)

// Driver はオーバーワールド帯の実行時状態と操作をまとめる。DungeonState が保持し委譲する。
// オーバーワールドとダンジョンの本質的な違いは「フロアを作り直さず帯をスライドさせ続ける」点だけで、
// その帯固有のロジックをこの Driver に閉じ込め、states パッケージから分離する。
type Driver struct {
	planner mapplanner.PlannerType
	// definition は帯形状の供給元。新規開始で使い、ロード復元では帯形状を SeamlessBand から得るので不要
	definition *dungeon.OverworldDefinition
	params   *NewGameParams // 新規開始のプレイ固有パラメータ。ロード復元では nil
	band     *worldstream.Band
	gen      worldstream.ChunkGen
	frontCfg worldstream.FrontConfig
}

// NewDriver は帯ドライバを構成する。params が非 nil なら新規開始、nil ならロード復元。
// definition は新規開始時の帯形状の供給元。ロード復元では帯形状を SeamlessBand から得るので nil でよい。
// 実際の帯生成・復元は Start で行う。
func NewDriver(planner mapplanner.PlannerType, definition *dungeon.OverworldDefinition, params *NewGameParams) *Driver {
	return &Driver{planner: planner, definition: definition, params: params}
}

// Start は帯ドライバを用意する。新規開始なら初期帯を生成し現ステージをオーバーワールドに
// 確定する。ロード復元なら SeamlessBand から Band と ChunkGen を作り直す。前線位置も初回
// 描画前に導出する。
func (dr *Driver) Start(world w.World) error {
	d := query.GetDungeon(world)

	// 視界の強制再計算は呼び出し側 DungeonState.OnStart が開始時にまとめて立てる。
	// オーバーワールドと通常ダンジョンで同じ扱いにするため、ここでは触らない。

	// 帯データは現ステージの StageField が持つ。ロード復元なら serde で戻っており、新規開始なら未生成で nil。
	sb := query.GetSeamlessBand(world)
	if sb != nil && sb.Active {
		// ロード復元。CurrentStage は serde で復元済みなので触らない。帯データは
		// オーバーワールドの StageField にしか無く、遺跡滞在中のセーブは現ステージが遺跡なので
		// ここには到達しない。newResumeStateFactory が DungeonState を選ぶ。
		if err := dr.restoreFromSave(world, sb); err != nil {
			return err
		}
	} else {
		// 新規開始。オーバーワールドから始める。共存機構が現在地を識別するのに使う。
		d.CurrentStage = gc.NewOverworldStage()
		if err := dr.startNewBand(world); err != nil {
			return err
		}
	}

	// 前線の現在位置を初回フレームの描画前に確定させる。Update を待つと最初の1フレーム
	// FrontEastAbsX が未初期化になりうるため、ここで一度導出しておく
	dr.UpdateFront(world)
	return nil
}

// restoreFromSave はセーブ済みの SeamlessBand から Band ドライバと ChunkGen を再構築する。
// 帯タイル・Level・プレイヤーは serde で復元済みなので再生成はしない。
func (dr *Driver) restoreFromSave(world w.World, sb *gc.SeamlessBand) error {
	dr.band = worldstream.NewBandAt(sb.ChunkW, sb.K, sb.EastIndex)
	dr.gen = NewChunkGen(world, sb.RunSeed, sb.ChunkW, sb.ChunkH, dr.planner)
	dr.frontCfg = frontCfgFromBand(sb)
	query.InvalidateSpatialIndex(world)
	return nil
}

// frontCfgFromBand は永続状態から寒波前線の前進パラメータを復元する。
func frontCfgFromBand(sb *gc.SeamlessBand) worldstream.FrontConfig {
	return worldstream.FrontConfig{
		StartEast:    sb.Front.StartAbsX,
		ColdWidth:    sb.Front.ColdWidth,
		AdvanceTurns: sb.Front.AdvanceTurns,
		Step:         sb.Front.Step,
	}
}

// front は総経過ターン数から寒波前線の現在位置を導出する。
func (dr *Driver) front(world w.World) worldstream.Front {
	totalTurns := query.GetGameTime(world).TotalTurns
	return worldstream.FrontAt(dr.frontCfg, totalTurns)
}

// startNewBand は新規開始として初期帯を決定的生成し、帯状態を SeamlessBand へ記録し、
// プレイヤーを中央チャンクへ置き、開始チャンクに遺跡入口を置く。帯パラメータは params から取る。
func (dr *Driver) startNewBand(world w.World) error {
	p := dr.params
	if p == nil {
		return fmt.Errorf("新規オーバーワールドの開始には帯パラメータが必要")
	}
	if dr.definition == nil {
		return fmt.Errorf("新規オーバーワールドの開始には帯形状の定義が必要")
	}
	// 帯形状はマスタ、すなわち OverworldDefinition から取る。RunSeed だけがプレイ固有
	chunkW, chunkH, k := dr.definition.BandShape()
	dr.band = worldstream.NewBand(chunkW, k)
	dr.gen = NewChunkGen(world, p.RunSeed, chunkW, chunkH, dr.planner)

	// 帯データを現ステージ、すなわちオーバーワールドの StageField エンティティへ確保する。
	// 以後この帯データの有無がオーバーワールド判定を兼ねる。値を書き込んでセーブに対応する
	sb := query.EnsureSeamlessBand(world)
	sb.Active = true
	sb.RunSeed = p.RunSeed
	sb.EastIndex = dr.band.EastIndex()
	sb.ChunkW = chunkW
	sb.ChunkH = chunkH
	sb.K = dr.band.K()

	// 寒波前線を初期化する。極低温ゾーン東端を西チャンクの東端（プレイヤーの1チャンク背後）に置く。
	// これで開始時からプレイヤーの背後に霜が見え、西へ戻ると凍える。以東へ進み帯がシフトすると前線は
	// 絶対軸に留まるため背後へ離れていく。普通に東進する限り触れない遅い地平にする。
	dr.frontCfg = worldstream.FrontConfig{
		StartEast:    worldstream.BandOriginX(dr.band.EastIndex(), chunkW) + consts.AbsTileX(chunkW),
		ColdWidth:    frontColdWidthChunks.Tiles(chunkW),
		AdvanceTurns: frontAdvanceTurns,
		Step:         frontStep,
	}
	sb.Front.Active = true
	sb.Front.StartAbsX = dr.frontCfg.StartEast
	sb.Front.ColdWidth = dr.frontCfg.ColdWidth
	sb.Front.AdvanceTurns = dr.frontCfg.AdvanceTurns
	sb.Front.Step = dr.frontCfg.Step

	// 初期帯 ＝ K*chunkW × chunkH の単一マップを決定的生成する。探索履歴はStageField が持ち初期化済み
	if err := dr.generateBandChunks(world, chunkW, chunkH); err != nil {
		return err
	}

	// プレイヤーを中央チャンクの中央へ。居なければ生成、居れば移動
	cx := (dr.band.K() / 2).Tiles(chunkW) + chunkW/2
	cy := chunkH / 2
	if _, err := query.GetPlayerEntity(world); err != nil {
		if _, serr := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: cx, Y: cy}, "Ash"); serr != nil {
			return fmt.Errorf("プレイヤー生成失敗: %w", serr)
		}
	} else if merr := lifecycle.MovePlayerToPosition(world, consts.Coord[consts.Tile]{X: cx, Y: cy}); merr != nil {
		return fmt.Errorf("プレイヤー配置失敗: %w", merr)
	}

	// 開始チャンクに遺跡入口を1つ置く。プレイヤーの数タイル東、歩いて到達できる位置。
	// 触れて Enter で遺跡へ入れる
	if _, err := lifecycle.SpawnDungeonEntrance(world, cx+2, cy, dungeon.DungeonForest.Name()); err != nil {
		return fmt.Errorf("遺跡入口の配置に失敗: %w", err)
	}

	// 開始チャンクに街を配置する。プレイヤー開始位置を中心に店・雇用・合成・収納を置く。
	// 街はオーバーワールドの地物なので、新規ゲームはこの街から始まり TownState を経由しない
	if err := spawnTown(world, consts.Coord[consts.Tile]{X: cx, Y: cy}); err != nil {
		return fmt.Errorf("街の配置に失敗: %w", err)
	}

	query.InvalidateSpatialIndex(world)
	return nil
}

// syncBandState は Band の現在 eastIndex を Dungeon の永続状態へ書き戻す。これでセーブに反映される。
func (dr *Driver) syncBandState(world w.World) {
	query.GetSeamlessBand(world).EastIndex = dr.band.EastIndex()
}

// generateBandChunks は Level を帯全幅に設定し、K チャンクを各スロットへ決定的生成する。
// Level 設定は帯幅が不変なので再設定しても冪等で無害。
func (dr *Driver) generateBandChunks(world w.World, chunkW, chunkH consts.Tile) error {
	query.EnsureStageField(world, gc.NewOverworldStage()).Level = gc.Level{TileWidth: dr.band.Width(), TileHeight: chunkH}
	for i := range dr.band.K() {
		if err := dr.gen(i, i.Tiles(chunkW)); err != nil {
			return fmt.Errorf("チャンク生成失敗 (slot=%d): %w", i, err)
		}
	}
	return nil
}

// EastIndex は帯の現在の東インデックスを返す。テストや検証用。
func (dr *Driver) EastIndex() consts.Chunk {
	return dr.band.EastIndex()
}

// UpdateFront は総ターン数から導出した寒波前線の現在位置を永続状態へ反映する。
// 位置は導出値なので毎フレーム書いても冪等。描画や凍結効果はこの FrontEastAbsX を読む。
func (dr *Driver) UpdateFront(world w.World) {
	sb := query.GetSeamlessBand(world)
	if sb == nil || !sb.Front.Active {
		return
	}
	sb.Front.EastAbsX = dr.front(world).East
}

// MaybeShift はプレイヤーが中央チャンクを出ていれば帯をシフトし、シフトしたかを返す。
// シフトするとリベースでプレイヤーが中央へ動くため、呼び出し側はカメラを再センタリングする。
//
// 座標を平行移動する破壊的操作なので、ターンが完全に解決した安定点でのみ行う。すなわち
// プレイヤーターンの Player フェーズかつプレイヤーが継続アクティビティ中でないとき。
func (dr *Driver) MaybeShift(world w.World) (bool, error) {
	if query.GetTurnState(world).Phase != gc.TurnPhasePlayer {
		return false, nil
	}
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return false, fmt.Errorf("シフト判定にプレイヤーが必要: %w", err)
	}
	if query.HasActivity(world, playerEntity) {
		return false, nil
	}
	// 中央チャンクに収まるまでシフトを繰り返す。各シフトはプレイヤーを chunkW ぶん中央へ寄せるため、
	// 必ず有限回で収束する。
	shifted := false
	for {
		localX := world.Components.GridElement.Get(playerEntity).X
		if dr.band.ShouldShiftEast(localX) {
			if err := dr.band.ShiftEast(world, dr.gen); err != nil {
				return shifted, err
			}
			dr.syncBandState(world)
			shifted = true
			continue
		}
		// 西シフトは寄り道からの復帰時のみ。ラン開始の eastIndex=0 より西には何も生成されて
		// いないため、eastIndex を負にする西シフトは行わない。プレイヤーは帯西端で自然に止まる
		if dr.band.ShouldShiftWest(localX) && dr.band.EastIndex() > 0 {
			if err := dr.band.ShiftWest(world, dr.gen); err != nil {
				return shifted, err
			}
			dr.syncBandState(world)
			shifted = true
			continue
		}
		break
	}
	return shifted, nil
}
