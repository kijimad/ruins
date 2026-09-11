package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnimationSystem_String はシステム名を固定する。
func TestAnimationSystem_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "AnimationSystem", NewAnimationSystem().String())
}

// TestAnimationSystem_Update_無効なら何もしない は、DisableAnimation のとき SpriteKey を
// 更新しないことを固定する。
func TestAnimationSystem_Update_無効なら何もしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.DisableAnimation = true
	e := world.ECS.NewEntity()
	world.Components.SpriteRender.Add(e, &gc.SpriteRender{AnimKeys: []string{"f0", "f1"}, SpriteKey: "orig"})

	sys := NewAnimationSystem()
	require.NoError(t, sys.Update(world))

	assert.Equal(t, "orig", world.Components.SpriteRender.Get(e).SpriteKey, "無効時はキーが変わらない")
}

// TestAnimationSystem_Update_アニメキーからSpriteKeyを更新する は、アニメーション有効時に
// フレームのキーへ更新することを固定する。1回更新なら先頭フレームになる。
func TestAnimationSystem_Update_アニメキーからSpriteKeyを更新する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.DisableAnimation = false
	e := world.ECS.NewEntity()
	world.Components.SpriteRender.Add(e, &gc.SpriteRender{AnimKeys: []string{"f0", "f1"}, SpriteKey: "orig"})

	sys := NewAnimationSystem()
	require.NoError(t, sys.Update(world))

	// counter=1, numFrames=2, frameInterval=60, frameIndex=(1/60)%2=0
	assert.Equal(t, "f0", world.Components.SpriteRender.Get(e).SpriteKey, "先頭フレームのキーになる")
}

// TestAnimationSystem_Update_アニメキーが空なら変えない は、AnimKeys が空の SpriteRender を
// 更新対象から外すことを固定する。
func TestAnimationSystem_Update_アニメキーが空なら変えない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.DisableAnimation = false
	e := world.ECS.NewEntity()
	world.Components.SpriteRender.Add(e, &gc.SpriteRender{SpriteKey: "orig"})

	sys := NewAnimationSystem()
	require.NoError(t, sys.Update(world))

	assert.Equal(t, "orig", world.Components.SpriteRender.Get(e).SpriteKey, "AnimKeysが空なら変えない")
}
