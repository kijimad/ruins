package vrt

import (
	"sync"
	"testing"

	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/stretchr/testify/require"
)

var (
	sharedUIResources *resources.UIResources
	uiResourcesOnce   sync.Once
	errUIResources    error
)

// SharedUIResources はテスト用UIリソースを返す。sync.Onceで1回だけ初期化する
func SharedUIResources(t *testing.T) resources.UIResources {
	t.Helper()
	uiResourcesOnce.Do(func() {
		fonts, err := loader.LoadFonts()
		if err != nil {
			errUIResources = err
			return
		}
		uir, err := loader.LoadUIResources(fonts)
		if err != nil {
			errUIResources = err
			return
		}
		sharedUIResources = &uir
	})

	require.NoError(t, errUIResources)
	return *sharedUIResources
}
