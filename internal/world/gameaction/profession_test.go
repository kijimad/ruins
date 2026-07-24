package gameaction

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProfession はテスト用の職業定義を作る
func newTestProfession() oapi.Profession {
	skills := []oapi.ProfessionSkill{
		{Id: string(gc.SkillSword), Value: 5},
	}
	return oapi.Profession{
		Id:   "test-profession",
		Name: "テスト職業",
		Abilities: oapi.Abilities{
			Strength:  10,
			Sensation: 11,
			Dexterity: 12,
			Agility:   13,
			Vitality:  14,
			Defense:   15,
		},
		Skills: &skills,
		Items: []oapi.ProfessionItem{
			{Name: "木の棒", Count: 3},
		},
		Equips: []oapi.ProfessionEquip{
			{Name: "木刀", Slot: oapi.EquipSlotWEAPON1},
		},
	}
}

func TestApplyProfession_成功時に能力値スキル装備アイテムが反映される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "Ash")
	require.NoError(t, err)

	prof := newTestProfession()
	err = ApplyProfession(world, player, prof)
	require.NoError(t, err)

	profComp := world.Components.Profession.Get(player)
	assert.Equal(t, "test-profession", profComp.ID, "職業IDが反映されるべき")

	abils := world.Components.Abilities.Get(player)
	assert.Equal(t, 10, abils.Strength.Base, "筋力が反映されるべき")
	assert.Equal(t, 11, abils.Sensation.Base, "感覚が反映されるべき")
	assert.Equal(t, 12, abils.Dexterity.Base, "器用さが反映されるべき")
	assert.Equal(t, 13, abils.Agility.Base, "敏捷性が反映されるべき")
	assert.Equal(t, 14, abils.Vitality.Base, "体力が反映されるべき")
	assert.Equal(t, 15, abils.Defense.Base, "防御力が反映されるべき")

	skills := world.Components.Skills.Get(player)
	assert.Equal(t, 5, skills.Get(gc.SkillSword).Value, "職業のスキル初期値が反映されるべき")

	assert.True(t, world.Components.CharModifiers.Has(player), "CharModifiersが再計算され付与されるべき")

	_, ok := query.FindStackableInInventory(world, "木の棒")
	assert.True(t, ok, "初期アイテムがバックパックに生成されるべき")

	weapons := query.GetWeapons(world, player)
	require.NotNil(t, weapons[0], "WEAPON1スロットに初期装備が装備されるべき")
	name := world.Components.Name.Get(*weapons[0])
	assert.Equal(t, "木刀", name.Name, "指定した装備アイテムがWEAPON1に装備されるべき")
}

func TestApplyProfession_職業を再適用すると職業IDが上書きされる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "Ash")
	require.NoError(t, err)

	first := newTestProfession()
	require.NoError(t, ApplyProfession(world, player, first))

	second := newTestProfession()
	second.Id = "second-profession"
	second.Items = nil
	second.Equips = nil
	require.NoError(t, ApplyProfession(world, player, second))

	profComp := world.Components.Profession.Get(player)
	assert.Equal(t, "second-profession", profComp.ID, "再適用時は既存のProfessionが更新されるべき")
}

func TestApplyProfession_異常系(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mutate      func(*oapi.Profession)
		errContains string
	}{
		{
			name: "存在しない初期アイテム名でエラー",
			mutate: func(p *oapi.Profession) {
				p.Items = []oapi.ProfessionItem{{Name: "存在しないアイテム", Count: 1}}
				p.Equips = nil
			},
			errContains: "職業の初期アイテム生成に失敗",
		},
		{
			name: "存在しない初期装備アイテム名でエラー",
			mutate: func(p *oapi.Profession) {
				p.Items = nil
				p.Equips = []oapi.ProfessionEquip{{Name: "存在しないアイテム", Slot: oapi.EquipSlotWEAPON1}}
			},
			errContains: "職業の初期装備生成に失敗",
		},
		{
			name: "不正な装備スロット名でエラー",
			mutate: func(p *oapi.Profession) {
				p.Items = nil
				p.Equips = []oapi.ProfessionEquip{{Name: "木刀", Slot: "BOGUS"}}
			},
			errContains: "不正な装備スロット名",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 0, Y: 0}, "Ash")
			require.NoError(t, err)

			prof := newTestProfession()
			tt.mutate(&prof)

			err = ApplyProfession(world, player, prof)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.errContains)
		})
	}
}
