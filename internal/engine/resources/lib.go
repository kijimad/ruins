package resources

import "fmt"

// ResourceProvider は抽象的なリソースプロバイダインターフェース
// 具体的な実装は lib/resources パッケージに移動
type ResourceProvider interface {
	GetScreenDimensions() (width, height int)
	SetScreenDimensions(width, height int)
}

// ResourceInitializer はリソースの初期化を行うインターフェース
type ResourceInitializer interface {
	InitializeResources() error
}

// Resources はジェネリクス型を使用した型安全な実装
type Resources[T ResourceInitializer] struct {
	Game T
}

// InitResources はジェネリクス型を使用した型安全な実装
func InitResources[T ResourceInitializer](gameResources T) (*Resources[T], error) {
	if err := gameResources.InitializeResources(); err != nil {
		return nil, fmt.Errorf("failed to initialize resources: %w", err)
	}

	return &Resources[T]{Game: gameResources}, nil
}
