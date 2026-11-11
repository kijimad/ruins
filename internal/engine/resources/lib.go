package resources

import (
	"errors"
	"fmt"
)

// errInitResources はリソースの初期化に失敗したときに返す
var errInitResources = errors.New("failed to initialize resources")

// ResourceProvider は抽象的なリソースプロバイダインターフェース
// 具体的な実装は internal/resources パッケージに移動
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
		return nil, fmt.Errorf("%w: %w", errInitResources, err)
	}

	return &Resources[T]{Game: gameResources}, nil
}
