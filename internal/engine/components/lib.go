package components

import (
	"fmt"

	"github.com/mlange-42/ark/ecs"
)

// ComponentInitializer はコンポーネントの初期化を行うインターフェース
type ComponentInitializer interface {
	InitializeComponents(world *ecs.World) error
}

// Components はジェネリクス型を使用した型安全な実装
type Components[T ComponentInitializer] struct {
	Game T
}

// InitComponents はジェネリクス型を使用した型安全な実装
func InitComponents[T ComponentInitializer](world *ecs.World, gameComponents T) (*Components[T], error) {
	if err := gameComponents.InitializeComponents(world); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return &Components[T]{Game: gameComponents}, nil
}
