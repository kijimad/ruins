package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockResourceInitializer struct {
	err        error
	initCalled bool
}

func (m *mockResourceInitializer) InitializeResources() error {
	m.initCalled = true
	return m.err
}

func TestInitResources_成功時にGameへ渡した値が設定される(t *testing.T) {
	t.Parallel()

	game := &mockResourceInitializer{}

	got, err := InitResources(game)

	require.NoError(t, err)
	require.NotNil(t, got)
	assert.True(t, game.initCalled)
	assert.Same(t, game, got.Game)
}

func TestInitResources_初期化エラーはラップして返す(t *testing.T) {
	t.Parallel()

	wantErr := assert.AnError
	game := &mockResourceInitializer{err: wantErr}

	got, err := InitResources(game)

	require.Error(t, err)
	assert.Nil(t, got)
	require.ErrorIs(t, err, wantErr)
	assert.ErrorContains(t, err, "failed to initialize resources")
}
