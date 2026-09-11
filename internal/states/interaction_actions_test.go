package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetInteractionActions_種別ごとにラベルを組む は、相互作用の種別ごとに
// アクションのラベルを1つ組むことを固定する。ラベルは query.T の翻訳を挟むため
// 中身の文字列は固定せず、非空であることと Interaction・Target の対応を見る。
// world は Ark が並行安全でないため各サブテストで作る。
func TestGetInteractionActions_種別ごとにラベルを組む(t *testing.T) {
	t.Parallel()

	withName := func(world w.World, e ecs.Entity) {
		world.Components.Name.Add(e, &gc.Name{Name: "対象"})
	}
	openDoor := func(world w.World, e ecs.Entity) {
		world.Components.Door.Add(e, &gc.Door{IsOpen: true})
	}
	closedDoor := func(world w.World, e ecs.Entity) {
		world.Components.Door.Add(e, &gc.Door{IsOpen: false})
	}
	noSetup := func(w.World, ecs.Entity) {}

	tests := []struct {
		name        string
		interaction gc.InteractionKind
		setup       func(w.World, ecs.Entity)
	}{
		{"閉じた扉は開く行を出す", gc.InteractionDoor, closedDoor},
		{"開いた扉は閉じる行を出す", gc.InteractionDoor, openDoor},
		{"会話", gc.InteractionTalk, withName},
		{"収納を調べる", gc.InteractionStorage, withName},
		{"近接攻撃", gc.InteractionMelee, withName},
		{"分解", gc.InteractionDisassemble, withName},
		{"次の階へワープ", gc.InteractionPortalNext, noSetup},
		{"前の階へワープ", gc.InteractionPortalPrev, noSetup},
		{"遺跡へ入る", gc.InteractionDungeonEnter, noSetup},
		{"キューブメニューを開く", gc.InteractionOpenCubeMenu, noSetup},
		{"運転", gc.InteractionDrive, noSetup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			e := world.ECS.NewEntity()
			tt.setup(world, e)

			actions := getInteractionActions(world,
				&gc.Interactable{Interactions: []gc.InteractionKind{tt.interaction}}, e, "→")

			require.Len(t, actions, 1, "1つのアクションを組む")
			assert.Equal(t, tt.interaction, actions[0].Interaction, "対応する種別を持つ")
			assert.Equal(t, e, actions[0].Target, "対象エンティティを指す")
			assert.NotEmpty(t, actions[0].Label, "ラベルは非空")
		})
	}

	t.Run("Doorコンポーネントが無ければ扉の行は出さない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		e := world.ECS.NewEntity() // Door を付けない

		actions := getInteractionActions(world,
			&gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionDoor}}, e, "→")

		assert.Empty(t, actions, "Door が無ければアクションは出ない")
	})
}
