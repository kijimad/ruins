package query

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetEntityID(t *testing.T) {
	t.Parallel()

	// Ark の world は並行安全でないので、並列サブテストごとに自前の world を作る
	t.Run("RawIDを持てばそのIDを返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: "sword"})
		assert.Equal(t, "sword", GetEntityID(e, world))
	})

	t.Run("RawIDを持たなければ空文字", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		assert.Empty(t, GetEntityID(e, world))
	})

	t.Run("死亡エンティティは空文字", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.RawID.Add(e, &gc.RawID{ID: "sword"})
		world.ECS.RemoveEntity(e)
		assert.Empty(t, GetEntityID(e, world))
	})
}

func TestNameMarkup(t *testing.T) {
	t.Parallel()

	// Ark の world は並行安全でないので、並列サブテストごとに自前の world を作る
	t.Run("プレイヤーはplayerタグ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.Player.Add(e, &gc.Player{})
		assert.Equal(t, gamelog.Tag("player", "アッシュ"), NameMarkup(e, "アッシュ", world))
	})

	t.Run("NPCはnpcタグ", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		world.Components.SoloAI.Add(e, &gc.SoloAI{})
		assert.Equal(t, gamelog.Tag("npc", "ゴブリン"), NameMarkup(e, "ゴブリン", world))
	})

	t.Run("それ以外は裸のテキスト", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity()
		assert.Equal(t, "木の板", NameMarkup(e, "木の板", world))
	})
}
