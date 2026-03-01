package worldhelper

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetActivity(t *testing.T) {
	t.Parallel()

	t.Run("アクティビティがない場合はnilを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		result := GetActivity(world, entity)
		assert.Nil(t, result)
	})

	t.Run("アクティビティがある場合は取得できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		activity := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
			TurnsTotal:   5,
			TurnsLeft:    5,
		}
		entity.AddComponent(world.Components.Activity, activity)

		result := GetActivity(world, entity)
		assert.NotNil(t, result)
		assert.Equal(t, gc.BehaviorWait, result.BehaviorName)
	})
}

func TestHasActivity(t *testing.T) {
	t.Parallel()

	t.Run("アクティビティがない場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		assert.False(t, HasActivity(world, entity))
	})

	t.Run("Running状態のアクティビティがある場合はtrue", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		activity := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
		}
		entity.AddComponent(world.Components.Activity, activity)

		assert.True(t, HasActivity(world, entity))
	})

	t.Run("Paused状態のアクティビティがある場合はfalse", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		activity := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStatePaused,
		}
		entity.AddComponent(world.Components.Activity, activity)

		assert.False(t, HasActivity(world, entity))
	})
}

func TestSetActivity(t *testing.T) {
	t.Parallel()

	t.Run("新規にアクティビティを設定できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		activity := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
		}
		SetActivity(world, entity, activity)

		result := GetActivity(world, entity)
		assert.NotNil(t, result)
		assert.Equal(t, gc.BehaviorWait, result.BehaviorName)
	})

	t.Run("既存のアクティビティを上書きできる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		// 最初のアクティビティを設定
		activity1 := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
		}
		entity.AddComponent(world.Components.Activity, activity1)

		// 新しいアクティビティで上書き
		activity2 := &gc.Activity{
			BehaviorName: gc.BehaviorRest,
			State:        gc.ActivityStateRunning,
		}
		SetActivity(world, entity, activity2)

		result := GetActivity(world, entity)
		assert.NotNil(t, result)
		assert.Equal(t, gc.BehaviorRest, result.BehaviorName)
	})
}

func TestRemoveActivity(t *testing.T) {
	t.Parallel()

	t.Run("アクティビティを削除できる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		activity := &gc.Activity{
			BehaviorName: gc.BehaviorWait,
			State:        gc.ActivityStateRunning,
		}
		entity.AddComponent(world.Components.Activity, activity)

		RemoveActivity(world, entity)

		result := GetActivity(world, entity)
		assert.Nil(t, result)
	})

	t.Run("アクティビティがない場合も安全に呼べる", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		entity := world.Manager.NewEntity()

		// パニックしないことを確認
		RemoveActivity(world, entity)

		result := GetActivity(world, entity)
		assert.Nil(t, result)
	})
}
