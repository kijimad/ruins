package world

import (
	c "github.com/kijimaD/ruins/internal/engine/components"
	r "github.com/kijimaD/ruins/internal/engine/resources"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// Generic は型安全なワールド型
type Generic[C c.ComponentInitializer, R r.ResourceInitializer] struct {
	Manager    *ecs.Manager
	Components *c.Components[C]
	Resources  *r.Resources[R]
}

// InitGeneric は型安全なワールド初期化
func InitGeneric[C c.ComponentInitializer, R r.ResourceInitializer](gameComponents C, gameResources R) (Generic[C, R], error) {
	manager := ecs.NewManager()
	components, err := c.InitComponents(manager, gameComponents)
	if err != nil {
		return Generic[C, R]{}, err
	}

	resources, err := r.InitResources(gameResources)
	if err != nil {
		return Generic[C, R]{}, err
	}

	return Generic[C, R]{
		Manager:    manager,
		Components: components,
		Resources:  resources,
	}, nil
}
