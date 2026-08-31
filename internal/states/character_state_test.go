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

func TestCharacterState_OnStartが成功し閲覧から始まる(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)

	require.NoError(t, state.OnStart(world))
	assert.False(t, state.equip.Active(), "初期状態は閲覧で装備選択は開いていない")
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
		Effects:     []gc.StatEffect{{Stat: gc.StatDexterity, Value: -4}, {Stat: gc.StatAgility, Value: -2}},
	})

	props, err := state.Fetch(world)
	require.NoError(t, err)

	var health statusTabData
	for _, tab := range props.InfoTabs {
		if tab.ID == tabHealth {
			health = tab
		}
	}
	var arms statusItemData
	for _, it := range health.Items {
		if it.BodyPart == gc.BodyPartArms {
			arms = it
		}
	}

	assert.Contains(t, arms.Value, "Fracture", "行の値に不調名が出る")
	assert.Contains(t, arms.Description, "broken bone", "詳細に概要説明を持つ")
	require.NotEmpty(t, arms.Details)
	assert.True(t, arms.Details[0].Header, "影響の先頭は不調の見出し")

	var foundDex, foundAgi, foundProgress, foundTended bool
	for _, d := range arms.Details {
		switch {
		case d.Label == "Dexterity" && d.Value == "-4":
			foundDex = true
		case d.Label == "Agility" && d.Value == "-2":
			foundAgi = true
		case d.Label == "Progress" && d.Value == "60%":
			foundProgress = true
		case d.Label == "Treatment" && d.Value == "Tended 150%":
			foundTended = true
		}
	}
	assert.True(t, foundProgress, "進行度をタイマーから出す")
	assert.True(t, foundTended, "治療済みと質を出す")
	assert.True(t, foundDex, "影響に器用のデバフが出る")
	assert.True(t, foundAgi, "影響に敏捷のデバフが出る")

	// 症状が見出し行に出るので、モーダルの見出しは部位名だけにして重複を避ける
	content := infoDetailContent(arms)
	assert.Equal(t, "Arm", content.Name, "見出しは部位名のみで症状を重ねない")
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
