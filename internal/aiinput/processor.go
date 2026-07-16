package aiinput

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"

	"github.com/mlange-42/ark/ecs"
)

// activationMargin は視界半径に加える余白タイル数。
// 圏外の敵が視界に入るのと同じターンに反応できるよう、視界より一回り広く処理する
const activationMargin = 2

// activationRadius は SoloAI を距離カリングせず処理するチェビシェフ半径（タイル）。
// この半径を超えた非交戦 SoloAI は行動計画をスキップする（アクティベーション半径）
const activationRadius = int(consts.VisionRadiusTiles) + activationMargin

// Processor はAIエンティティの行動処理を管理する。
// AI.Plannerフィールドに基づいてsoloPlannerとsquadPlannerを使い分ける
type Processor struct {
	logger   *logger.Logger
	planners map[gc.PlannerType]Planner
}

// NewProcessor は新しいProcessorを作成する。
// rngはゲーム全体のseedから派生した乱数生成器を渡す
func NewProcessor(rng *rand.Rand) *Processor {
	return &Processor{
		logger: logger.New(logger.CategoryTurn),
		planners: map[gc.PlannerType]Planner{
			gc.PlannerSolo:  newSoloPlanner(rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))),
			gc.PlannerSquad: newSquadPlanner(rand.New(rand.NewPCG(rng.Uint64(), rng.Uint64()))),
		},
	}
}

// ProcessAll は全AIエンティティを処理する。
// Soloを先に処理し、敵の移動結果を反映したSquadの判断を可能にする
func (p *Processor) ProcessAll(world w.World) error {
	if err := p.processByPlanner(world, gc.PlannerSolo); err != nil {
		return err
	}
	return p.processByPlanner(world, gc.PlannerSquad)
}

// processByPlanner は指定されたPlannerTypeを持つAIエンティティを処理する。
// SoloAI/SquadAI は別コンポーネントのため、種別に対応するコンポーネントで絞り込む
func (p *Processor) processByPlanner(world w.World, plannerType gc.PlannerType) error {
	planner := p.planners[plannerType]

	var targets []ecs.Entity
	switch plannerType {
	case gc.PlannerSquad:
		squadQuery := ecs.NewFilter2[gc.SquadAI, gc.GridElement](world.ECS).Query()
		for squadQuery.Next() {
			targets = append(targets, squadQuery.Entity())
		}
	case gc.PlannerSolo:
		soloQuery := ecs.NewFilter2[gc.SoloAI, gc.GridElement](world.ECS).Query()
		for soloQuery.Next() {
			targets = append(targets, soloQuery.Entity())
		}
		// クエリ反復を終えてからカリングする。反復中に GetPlayerEntity の別クエリを張らない
		culled, err := cullDistantSolo(world, targets)
		if err != nil {
			return err
		}
		targets = culled
	}

	for _, entity := range targets {
		if world.Components.Dead.Has(entity) {
			continue
		}
		runAPLoop(world, entity, planner, p.logger)
	}

	return nil
}

// cullDistantSolo は遠方の非交戦 SoloAI を処理対象から除外する。
// 交戦中は視界外でも対象を追い続ける設計のため、距離に関わらず残す。
func cullDistantSolo(world w.World, targets []ecs.Entity) ([]ecs.Entity, error) {
	playerEntity, err := query.GetPlayerEntity(world)
	if err != nil {
		return nil, fmt.Errorf("カリングにはプレイヤーが必要: %w", err)
	}
	if !world.Components.GridElement.Has(playerEntity) {
		return nil, fmt.Errorf("プレイヤーに位置情報がありません")
	}
	playerGrid := world.Components.GridElement.Get(playerEntity)
	px, py := int(playerGrid.X), int(playerGrid.Y)

	kept := make([]ecs.Entity, 0, len(targets))
	for _, entity := range targets {
		solo := world.Components.SoloAI.Get(entity)
		if solo != nil && !isActiveCombatState(solo.SubState) {
			grid := world.Components.GridElement.Get(entity)
			if geometry.ChebyshevDistance(px, py, int(grid.X), int(grid.Y)) > activationRadius {
				// 圏外の待機・徘徊敵はスキップ。画面外で観測不能なため凍結してよい
				continue
			}
		}
		kept = append(kept, entity)
	}
	return kept, nil
}

// isActiveCombatState は交戦中（追跡・逃亡）状態かを返す。
// 交戦中の敵は視界外でも追い続ける設計のため距離カリングの対象外とする
func isActiveCombatState(s gc.AIStateSubState) bool {
	return s == gc.AIStateChasing || s == gc.AIStateFleeing
}
