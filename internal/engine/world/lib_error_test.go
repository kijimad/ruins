package world

import (
	"errors"
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/require"
)

var errInit = errors.New("init failed")

// failComponents は InitializeComponents で失敗する初期化子。
type failComponents struct{}

func (failComponents) InitializeComponents(*ecs.World) error { return errInit }

// okComponents は成功する初期化子。リソース側の失敗を試すために使う。
type okComponents struct{}

func (okComponents) InitializeComponents(*ecs.World) error { return nil }

// failResources は InitializeResources で失敗する初期化子。
type failResources struct{}

func (failResources) InitializeResources() error { return errInit }

// okResources は成功する初期化子。コンポーネント側の失敗を試すために使う。
type okResources struct{}

func (okResources) InitializeResources() error { return nil }

// TestInitGeneric_コンポーネント初期化の失敗を伝播する は、コンポーネント初期化で
// エラーが出たとき InitGeneric がそれを包んで返すことを固定する。
func TestInitGeneric_コンポーネント初期化の失敗を伝播する(t *testing.T) {
	t.Parallel()

	_, err := InitGeneric(failComponents{}, okResources{})
	require.ErrorIs(t, err, errInit)
}

// TestInitGeneric_リソース初期化の失敗を伝播する は、リソース初期化で
// エラーが出たとき InitGeneric がそれを包んで返すことを固定する。
func TestInitGeneric_リソース初期化の失敗を伝播する(t *testing.T) {
	t.Parallel()

	_, err := InitGeneric(okComponents{}, failResources{})
	require.ErrorIs(t, err, errInit)
}
