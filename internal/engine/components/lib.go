package components

import (
	"fmt"

	ecs "github.com/x-hgg-x/goecs/v2"
)

// ComponentInitializer はコンポーネントの初期化を行うインターフェース
type ComponentInitializer interface {
	InitializeComponents(manager *ecs.Manager) error
}

// Components はジェネリクス型を使用した型安全な実装
type Components[T ComponentInitializer] struct {
	Game T
}

// InitComponents はジェネリクス型を使用した型安全な実装
func InitComponents[T ComponentInitializer](manager *ecs.Manager, gameComponents T) (*Components[T], error) {
	if err := gameComponents.InitializeComponents(manager); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	return &Components[T]{Game: gameComponents}, nil
}
