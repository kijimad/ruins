package world

import (
	c "github.com/kijimaD/ruins/internal/engine/components"
	r "github.com/kijimaD/ruins/internal/engine/resources"

	"github.com/mlange-42/ark/ecs"
)

// Generic は型安全なワールド型
type Generic[C c.ComponentInitializer, R r.ResourceInitializer] struct {
	World      *ecs.World
	Components *c.Components[C]
	Resources  *r.Resources[R]
}

// InitGeneric は型安全なワールド初期化
func InitGeneric[C c.ComponentInitializer, R r.ResourceInitializer](gameComponents C, gameResources R) (Generic[C, R], error) {
	world := ecs.NewWorld()
	components, err := c.InitComponents(world, gameComponents)
	if err != nil {
		return Generic[C, R]{}, err
	}

	resources, err := r.InitResources(gameResources)
	if err != nil {
		return Generic[C, R]{}, err
	}

	return Generic[C, R]{
		World:      world,
		Components: components,
		Resources:  resources,
	}, nil
}
