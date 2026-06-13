package vrt

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/kijimaD/ruins/internal/loader"
	"github.com/kijimaD/ruins/internal/resources"
	"github.com/stretchr/testify/require"
)

var (
	sharedUIResources *resources.UIResources
	uiResourcesOnce   sync.Once
	uiResourcesErr    error
)

// SharedUIResources はテスト用UIリソースを返す。sync.Onceで1回だけ初期化する
func SharedUIResources(t *testing.T) resources.UIResources {
	t.Helper()
	uiResourcesOnce.Do(func() {
		fonts, err := loader.LoadFonts()
		if err != nil {
			uiResourcesErr = fmt.Errorf("フォント読み込みに失敗: %w", err)
			return
		}

		dougenzaka := fonts["dougenzaka"]
		nerd := fonts["nerd"]
		fontSources := []*text.GoTextFaceSource{
			dougenzaka.FaceSource,
			nerd.FaceSource,
		}

		uir, err := resources.NewUIResources(fontSources)
		if err != nil {
			uiResourcesErr = fmt.Errorf("UIリソース初期化に失敗: %w", err)
			return
		}
		sharedUIResources = &uir
	})

	require.NoError(t, uiResourcesErr)
	return *sharedUIResources
}
