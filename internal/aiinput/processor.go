package aiinput

import (
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/mlange-42/ark/ecs"
)

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

	// runAPLoopがSetActivity等の構造変更を行うため、対象を集めてから反復後に処理する
	var targets []ecs.Entity
	if plannerType == gc.PlannerSquad {
		squadQuery := ecs.NewFilter2[gc.SquadAI, gc.GridElement](world.World).Query()
		for squadQuery.Next() {
			targets = append(targets, squadQuery.Entity())
		}
	} else {
		soloQuery := ecs.NewFilter2[gc.SoloAI, gc.GridElement](world.World).Query()
		for soloQuery.Next() {
			targets = append(targets, soloQuery.Entity())
		}
	}

	for _, entity := range targets {
		if world.Components.Dead.Has(entity) {
			continue
		}
		runAPLoop(world, entity, planner, p.logger)
	}

	return nil
}
