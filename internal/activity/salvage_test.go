package activity

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolDamageMultiplier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		weapon string
		prop   string
		want   int
	}{
		{"斧で木を殴るとグレード2で3倍", "axe", "tree_a", 3},
		{"モンキーレンチで木箱を殴るとグレード1で2倍", "monkey_wrench", "wooden_crate", 2},
		{"斧で木箱を殴っても分類が違うので1倍", "axe", "wooden_crate", 1},
		{"道具が必須の木を松明で殴ると0倍", "torch", "tree_a", 0},
		{"道具が必須でない木箱を松明で殴ると1倍", "torch", "wooden_crate", 1},
		{"分解定義の無いpropは1倍", "axe", "grass", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			require.NoError(t, lifecycle.UnequipAll(world, player))
			weapon, err := lifecycle.SpawnBackpackItem(world, tt.weapon, 1)
			require.NoError(t, err)
			lifecycle.MoveToEquip(world, weapon, player, gc.SlotWeapon1)
			query.GetWeaponSelection(world).Slot = 1
			target, err := lifecycle.SpawnProp(world, tt.prop, 10, 9)
			require.NoError(t, err)

			assert.Equal(t, tt.want, toolDamageMultiplier(world, player, target))
		})
	}
}

func TestSalvageRoll(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		weapon       string
		prop         string
		wantGrade    int
		wantFull     bool
		wantWithTool bool
	}{
		{"斧で木を壊すと全枠を出し道具を使う", "axe", "tree_a", 2, true, true},
		{"モンキーレンチで木箱を壊すと全枠を出し道具を使う", "monkey_wrench", "wooden_crate", 1, true, true},
		{"斧で木箱を壊すと分類が違うので回収だけ", "axe", "wooden_crate", 0, false, false},
		{"松明で木箱を壊すと回収だけ", "torch", "wooden_crate", 0, false, false},
		{"道具の分類が無い灌木は松明でも全枠を出す", "torch", "bush", 0, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			require.NoError(t, lifecycle.UnequipAll(world, player))
			weapon, err := lifecycle.SpawnBackpackItem(world, tt.weapon, 1)
			require.NoError(t, err)
			lifecycle.MoveToEquip(world, weapon, player, gc.SlotWeapon1)
			query.GetWeaponSelection(world).Slot = 1
			def, ok := raw.FindDisassembly(world.Resources.RawMaster, tt.prop)
			require.True(t, ok)

			grade, full, withTool := salvageRoll(world, player, def)

			assert.Equal(t, tt.wantGrade, grade)
			assert.Equal(t, tt.wantFull, full)
			assert.Equal(t, tt.wantWithTool, withTool)
		})
	}
}

func TestApplyAttackDamage_壊すと産出を落とす(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		weapon        string
		prop          string
		wantDrop      string
		wantMechanics bool
	}{
		{"斧で木を壊すと薪を落とすが機械スキルは伸びない", "axe", "tree_a", "firewood", false},
		{"モンキーレンチで木箱を壊すと機械スキルが伸びる", "monkey_wrench", "wooden_crate", "", true},
		{"松明で灌木を壊すと枝を落とし機械スキルは伸びない", "torch", "bush", "branch", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
			require.NoError(t, err)
			require.NoError(t, lifecycle.UnequipAll(world, player))
			weapon, err := lifecycle.SpawnBackpackItem(world, tt.weapon, 1)
			require.NoError(t, err)
			lifecycle.MoveToEquip(world, weapon, player, gc.SlotWeapon1)
			query.GetWeaponSelection(world).Slot = 1
			target, err := lifecycle.SpawnProp(world, tt.prop, 10, 9)
			require.NoError(t, err)
			attack, name, err := getAttackParams(player, world)
			require.NoError(t, err)
			expBefore := world.Components.Skills.Get(player).Get(gc.SkillMechanic).Exp.Current

			// 命中率の補正を大きく取り、壊れるまで必ず命中させる
			for range 20 {
				require.NoError(t, applyAttackDamage(player, target, world, attack, name, 1000, 0))
				if world.Components.Dead.Has(target) {
					break
				}
			}
			require.True(t, world.Components.Dead.Has(target))

			if tt.wantDrop != "" {
				drops := query.PickablesAt(world, consts.Coord[consts.Tile]{X: 10, Y: 9})
				require.NotEmpty(t, drops, "確定枠があるので必ず落とす")
				for _, d := range drops {
					assert.Equal(t, tt.wantDrop, query.GetEntityID(d, world))
				}
			}
			expAfter := world.Components.Skills.Get(player).Get(gc.SkillMechanic).Exp.Current
			assert.Equal(t, tt.wantMechanics, expAfter > expBefore)
		})
	}
}

func TestYieldsMarkup_アイテムを現在言語の表示名で並べる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetUserSettings(world).Language = "ja"

	got := yieldsMarkup([]lifecycle.YieldStack{{Name: "firewood", Count: 2}, {Name: "branch", Count: 1}}, world)
	assert.Equal(t, "<item>薪</item> x2、<item>枝</item> x1", got)
}
