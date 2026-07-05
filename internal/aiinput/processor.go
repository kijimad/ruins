package aiinput

import (
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/logger"
	w "github.com/kijimaD/ruins/internal/world"

	ecs "github.com/x-hgg-x/goecs/v2"
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

// processByPlanner は指定されたPlannerTypeを持つAIエンティティを処理する
func (p *Processor) processByPlanner(world w.World, plannerType gc.PlannerType) error {
	planner := p.planners[plannerType]

	world.Manager.Join(
		world.Components.AI,
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		ai := world.Components.AI.Get(entity).(*gc.AI)
		if ai.Planner != plannerType {
			return
		}
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		runAPLoop(world, entity, planner, p.logger)
	}))

	return nil
}
