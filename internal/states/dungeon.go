package states

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/dungeon"
	es "github.com/kijimaD/ruins/internal/engine/states"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/widgets/theme"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/world/stage"
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

	// planner・newGame・driver・overworldDefinition はオーバーワールドモードのときだけ使う。
	// 帯固有のロジックは overworld.Driver に閉じ込め、DungeonState は保持と委譲だけ行う
	planner mapplanner.PlannerType
	newGame *overworld.NewGameParams // 新規開始の帯パラメータ。ロード復元では nil
	driver  *overworld.Driver        // OnStart で構成する帯ドライバ。通常ダンジョンでは nil
	// overworldDefinition はオーバーワールドの種別。非 nil ならこの State は帯モード。
	// 種別を State が直接持つことで、登録表に無いテスト用の種別も注入できる
	overworldDefinition *dungeon.OverworldDefinition
}

// isSeamless はこの State がオーバーワールド帯モードかを返す。オーバーワールドとダンジョンの
// 本質的な違いは帯の有無で、それは OverworldDefinition 種別を持つかで表す。フラグでなく型で判定する。
func (st DungeonState) isSeamless() bool {
	return st.overworldDefinition != nil
}

// NewOverworldState はオーバーワールド探索ステートのファクトリを返す。
//
// オーバーワールドは帯を持つステージ種別 OverworldDefinition で、専用の State 型は持たず DungeonState
// として動く。帯固有のロジックは overworld.Driver に閉じ込めてあり、DungeonState は OnStart で
// ドライバを構成して開始を委譲し、Update でシフトを委譲するだけ。
//
// kind は帯形状の供給元。本番は登録済みの dungeon.DungeonOverworld を渡す。
// params が非 nil なら新規開始として初期帯を生成する。nil ならセーブからの復元とみなし、
// 帯形状は Driver の Start がオーバーワールドの StageField の SeamlessBand から読み取って再構築する。
func NewOverworldState(planner mapplanner.PlannerType, kind *dungeon.OverworldDefinition, params *overworld.NewGameParams) es.StateFactory[w.World] {
	return func() (es.State[w.World], error) {
		return &DungeonState{
			// overworldDefinition を持たせることで OnStart が帯モードへ分岐する
			DefinitionName:      kind.Name(),
			planner:             planner,
			newGame:             params,
			overworldDefinition: kind,
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

	// 開始時に視界を一度だけ強制再計算させる。VisionSystem は現ステージが変わらないと
	// キャッシュを無効化しないが、オーバーワールドは常に同一ステージ、通常ダンジョンの
	// ロード復帰も保存前と同じ現ステージで、いずれも自動再計算が働かない。加えて serde は
	// VisionState を空で復元するため、放置すると空の VisibleTiles のまま真っ暗になる。
	// オーバーワールドと通常ダンジョンで同じ扱いにするため、分岐前のここで立てる。
	query.GetVisionState(world).NeedsForceUpdate = true

	// Seamless なオーバーワールドは帯ドライバを構成して委譲する。帯固有のロジックは
	// overworld.Driver に閉じ込め、DungeonState はここで開始を委譲するだけにする
	if st.isSeamless() {
		st.driver = overworld.NewDriver(st.planner, st.overworldDefinition, st.newGame)
		return st.driver.Start(world)
	}

	// 進入先の遺跡定義名を決める。State に明示指定があればそれを使い、無ければ現ステージ、
	// すなわち今いる遺跡の名前を引き継ぐ。ダンジョン定義名は CurrentStage.Name から導出する。
	defName := st.DefinitionName
	if defName == "" {
		defName = query.GetDungeon(world).CurrentStage.Name
	}
	// ダンジョン種別を取得する。ここは Seamless 判定を抜けた通常ダンジョンなので DungeonDefinition のはず
	def, err := resolveDungeonDefinition(defName)
	if err != nil {
		return err
	}
	// 単一フロアを新規生成して現ステージに確定する。初回進入や golden の単発描画で使う。
	// これは共存を作らない: 他ステージの suspend も上り階段の結線もしないので、ゲーム中の
	// 階層移動(地上⇄遺跡・階の上り下り)には使わないこと。それらは enterDungeon/descend の
	// SwapTo を通し、退避と結線を伴って往復する。
	// 復帰モードでは再生成せず、復元済みの地形・エンティティ・プレイヤー位置をそのまま使う
	if !st.Resume {
		key := dungeonStageKey(defName, st.Depth)
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
	titleText := def.Name()
	if st.Depth > 0 {
		titleText = fmt.Sprintf("%s %dF", def.Name(), st.Depth)
	}
	splashFace := world.Resources.UIResources.Text.SplashFontFace
	titleEffect := gc.NewSplashTextEffect(titleText, splashFace, screenW, screenH)
	titleEntity := world.ECS.NewEntity()
	world.Components.VisualEffects.Add(titleEntity, &gc.VisualEffects{
		Effects: []gc.VisualEffect{titleEffect},
	})

	return nil
}

// OnStop はステートが停止される際に呼ばれる。
//
// 共存方式ではオーバーワールドと遺跡が同一 world に共存し、退避中ステージも保持するため、
// ここでは何もしない。world を捨てるのはタイトルへ戻る・ロードのときで、MainMenuState.OnStart
// の全 entity 削除と save の ECS.Reset が担う。ステージ単位の破棄が要る場合は stage.Purge を呼ぶ。
func (st *DungeonState) OnStop(_ w.World) error { return nil }

// checkPlayerDeath はプレイヤーの死亡状態をチェックする。Update フローの述語
func (st *DungeonState) checkPlayerDeath(world w.World) bool {
	playerDead := false
	playerDeadQuery := ecs.NewFilter2[gc.Player, gc.Dead](world.ECS).Query()
	for playerDeadQuery.Next() {
		playerDead = true
	}
	return playerDead
}

// Update はゲームステートの更新処理を行う
func (st *DungeonState) Update(world w.World) (es.Transition[w.World], error) {
	// 全ダンジョン踏破をオーバーワールド滞在時に判定する。判定条件は帯シフトと同じ
	// 「session保持かつ現ステージ深度0」。SetEventActive は冪等で視聴後は再発火しないので、
	// 毎フレーム呼んでも一度だけ発火する
	if st.driver != nil && query.IsOnOverworld(world) {
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
	// 現ステージがオーバーワールドのときだけ前線を進め帯をシフトする。帯ドライバは
	// オーバーワールド State だけが持ち、現ステージ深度0がオーバーワールドを表す。遺跡へ入ると
	// 同一 State 内で現ステージ深度が1以上へ変わり、そのあいだ帯を触らない。通常ダンジョンは
	// driver が nil で除外される。死亡やリクエスト遷移で早期 return したフレームも触らない
	if st.driver != nil && query.IsOnOverworld(world) && transition.Type == es.TransNone {
		st.driver.UpdateFront(world)
		shifted, serr := st.driver.MaybeShift(world)
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
