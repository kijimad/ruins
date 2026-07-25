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
	params     *NewGameParams // 新規開始のプレイ固有パラメータ。ロード復元では nil
	band       *worldstream.Band
	gen        worldstream.ChunkGen
	frontCfg   worldstream.FrontConfig
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
	// Rows がゼロ値なら1へ正規化して1行の帯として復元する
	rows := max(sb.Rows, 1)
	dr.band = worldstream.NewBandAt(sb.ChunkW, sb.ChunkH, sb.K, rows, sb.EastIndex)
	dr.gen = NewChunkGen(world, sb.RunSeed, sb.ChunkW, sb.ChunkH, rows, dr.planner)
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
	chunkW, chunkH, k, rows := dr.definition.BandShape()
	dr.band = worldstream.NewBand(chunkW, chunkH, k, rows)
	dr.gen = NewChunkGen(world, p.RunSeed, chunkW, chunkH, rows, dr.planner)

	// 帯データを現ステージ、すなわちオーバーワールドの StageField エンティティへ確保する。
	// 以後この帯データの有無がオーバーワールド判定を兼ねる。値を書き込んでセーブに対応する
	sb := query.EnsureSeamlessBand(world)
	sb.Active = true
	sb.RunSeed = p.RunSeed
	sb.EastIndex = dr.band.EastIndex()
	sb.ChunkW = chunkW
	sb.ChunkH = chunkH
	sb.K = dr.band.K()
	sb.Rows = dr.band.Rows()

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

	// プレイヤーを中央チャンク付近の歩行可能タイルへ湧かせる。開始チャンクの種別に依存せず、
	// 建物や遺跡入口の上でも壁を避けて安全に置く。居なければ生成、居れば移動
	cx := (dr.band.K() / 2).Tiles(chunkW) + chunkW/2
	cy := (dr.band.Rows() / 2).Tiles(chunkH) + chunkH/2
	spawn := walkableSpawnNear(world, cx, cy)
	if _, err := query.GetPlayerEntity(world); err != nil {
		if _, serr := lifecycle.SpawnPlayer(world, spawn, "Ash"); serr != nil {
			return fmt.Errorf("プレイヤー生成失敗: %w", serr)
		}
	} else if merr := lifecycle.MovePlayerToPosition(world, spawn); merr != nil {
		return fmt.Errorf("プレイヤー配置失敗: %w", merr)
	}

	query.InvalidateSpatialIndex(world)
	return nil
}

// walkableSpawnNear は (cx, cy) から外側のリングへ順に探し、BlockPass の無い最初のタイル座標を
// 返す。開始チャンクが荒れ地でなく建物や遺跡入口でも、プレイヤーを壁の中へ湧かせないための安全策。
// 帯は全域が dirt で埋まり歩行可能タイルが必ず近くにあるため、見つからなければ中央を返す。
func walkableSpawnNear(world w.World, cx, cy consts.Tile) consts.Coord[consts.Tile] {
	blocked := make(map[gc.GridElement]bool)
	q := query.ActiveFilter2[gc.GridElement, gc.BlockPass](world).Query()
	for q.Next() {
		blocked[*world.Components.GridElement.Get(q.Entity())] = true
	}
	x0, y0 := int(cx), int(cy)
	at := func(x, y int) consts.Coord[consts.Tile] {
		return consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
	}
	isBlocked := func(x, y int) bool { return blocked[gc.GridElement{Coord: at(x, y)}] }
	for r := range 100 {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				// リングの外周だけ見る。内側は前の r で確認済み
				if r > 0 && dx > -r && dx < r && dy > -r && dy < r {
					continue
				}
				if !isBlocked(x0+dx, y0+dy) {
					return at(x0+dx, y0+dy)
				}
			}
		}
	}
	return at(x0, y0)
}

// generateBandChunks は Level を帯全域に設定し、rows × K のチャンクを各スロットへ決定的生成する。
// Level 設定は帯寸法が不変なので再設定しても冪等で無害。
func (dr *Driver) generateBandChunks(world w.World, chunkW, chunkH consts.Tile) error {
	query.EnsureStageField(world, gc.NewOverworldStage()).Level = gc.Level{TileWidth: dr.band.Width(), TileHeight: dr.band.Height()}
	for cy := range dr.band.Rows() {
		for i := range dr.band.K() {
			c := worldstream.ChunkCoord{X: dr.band.EastIndex() + i, Y: cy}
			if err := dr.gen(c, i.Tiles(chunkW), cy.Tiles(chunkH)); err != nil {
				return fmt.Errorf("チャンク生成失敗 (x=%d, y=%d): %w", c.X, c.Y, err)
			}
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
			shifted = true
			continue
		}
		// 西シフトは寄り道からの復帰時のみ。ラン開始の eastIndex=0 より西には何も生成されて
		// いないため、eastIndex を負にする西シフトは行わない。プレイヤーは帯西端で自然に止まる
		if dr.band.ShouldShiftWest(localX) && dr.band.EastIndex() > 0 {
			if err := dr.band.ShiftWest(world, dr.gen); err != nil {
				return shifted, err
			}
			shifted = true
			continue
		}
		break
	}
	if shifted {
		// Band の最終 eastIndex を永続状態へ書き戻す。セーブに要るのは最終値だけなので、
		// シフトのたびでなくループを抜けてから一度だけ同期する
		query.GetSeamlessBand(world).EastIndex = dr.band.EastIndex()
	}
	return shifted, nil
}
