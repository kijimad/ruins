package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/overlay"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipableForSlot_所持する装備品が候補に出る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "iron_sword", 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "cloth_hat", 1)
	require.NoError(t, err)

	assert.NotEmpty(t, equipableForSlot(world, gc.SlotWeapon1), "武器を所持していれば武器スロットの候補に出る")
	assert.NotEmpty(t, equipableForSlot(world, gc.SlotHead), "頭防具を所持していれば頭スロットの候補に出る")
}

func TestCharacterState_OnStartが成功する(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)

	require.NoError(t, state.OnStart(world))
}

func TestCharacterState_装備スロットは武器5と防具7の合計12(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	props, err := state.Fetch(world)
	require.NoError(t, err)
	assert.Len(t, props.EquipSlots, 12, "武器5スロットと防具7スロット")
}

func TestCharacterState_情報タブは能力スキル効果健康基本の5つ(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	props, err := state.Fetch(world)
	require.NoError(t, err)
	labels := make([]string, len(props.InfoTabs))
	for i, tab := range props.InfoTabs {
		labels[i] = tab.Label
	}
	assert.Equal(t, []string{"Abilities", "Skills", "Effects", "Health", "Basic"}, labels)
}

func TestCharacterState_スキルタブはカテゴリ見出しを含む(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	props, err := state.Fetch(world)
	require.NoError(t, err)
	var skillTab statusTabData
	for _, tab := range props.InfoTabs {
		if tab.ID == tabSkills {
			skillTab = tab
		}
	}
	hasHeader := false
	for _, item := range skillTab.Items {
		if item.IsHeader {
			hasHeader = true
			break
		}
	}
	assert.True(t, hasHeader, "スキルタブはカテゴリ見出し行を持つ")
}

func TestCharacterState_健康タブは不調の概要と影響を詳細に持つ(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	world.Components.HealthStatus.Get(player).Parts[gc.BodyPartArms].SetCondition(gc.HealthCondition{
		Type:        gc.ConditionFracture,
		Timer:       60,
		Severity:    gc.SeverityMedium,
		TendQuality: 150,
	})

	props, err := state.Fetch(world)
	require.NoError(t, err)

	var health statusTabData
	for _, tab := range props.InfoTabs {
		if tab.ID == tabHealth {
			health = tab
		}
	}

	// 部位はカテゴリ見出し、症状は1エントリ1件
	var hasArmHeader bool
	var frac statusItemData
	for _, it := range health.Items {
		if it.BodyPart == gc.BodyPartArms && it.IsHeader {
			hasArmHeader = true
		}
		if it.BodyPart == gc.BodyPartArms && it.ConditionType == gc.ConditionFracture {
			frac = it
		}
	}
	assert.True(t, hasArmHeader, "部位はカテゴリ見出しになる")
	require.Equal(t, gc.ConditionFracture, frac.ConditionType, "腕の骨折が1エントリとして並ぶ")
	assert.Equal(t, "60%", frac.Value, "エントリの値に進行度")
	assert.Contains(t, frac.Label, "Tended 150%", "症状名の右に治療状態")

	// 詳細は選んだ1症状ぶんを組む。名前は症状名、概要と性能行を持つ
	content := healthDetailContent(world, frac)
	// 見出しは症状名のみ。重症度は Progress 行の%で表すので名前に付けない
	assert.Equal(t, "Fracture", content.Name, "見出しは症状名")
	assert.Contains(t, content.Desc, "broken bone", "概要説明を持つ")

	var prog, tend, pain, manip string
	for _, r := range content.Rows {
		switch r.Label {
		case "Progress":
			prog = r.Value
		case "Treatment":
			tend = r.Value
		case "Pain":
			pain = r.Value
		case "Manipulation":
			manip = r.Value
		}
	}
	assert.Equal(t, "60%", prog, "進行度をタイマーから出す")
	assert.Equal(t, "Tended 150%", tend, "治療済みと質")
	// 腕の骨折(中)は未治療なら痛み+36・操作-40。応急処置済みなので半減し痛み+18・操作-20
	assert.Equal(t, "+18", pain, "応急処置で痛みが半減する")
	assert.Equal(t, "-20", manip, "応急処置で機能低下が半減する")
}

func TestDetailPageCount_componentが多いレイガンは複数ページになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	entity, err := lifecycle.SpawnBackpackItem(world, "ray_gun", 1)
	require.NoError(t, err)
	assert.Greater(t, overlay.DetailPageCount(world, entity), 1, "性能区画が多いアイテムの詳細は複数ページに分割される")
}
