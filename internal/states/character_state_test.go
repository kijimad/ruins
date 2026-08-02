package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/widgets/menuscreen"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipableForSlot_所持する装備品が候補に出る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "鉄の剣", 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "布の帽子", 1)
	require.NoError(t, err)

	assert.NotEmpty(t, equipableForSlot(world, gc.SlotWeapon1), "武器を所持していれば武器スロットの候補に出る")
	assert.NotEmpty(t, equipableForSlot(world, gc.SlotHead), "頭防具を所持していれば頭スロットの候補に出る")
}

func TestCharacterState_OnStartで各マウントを初期化する(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)

	require.NoError(t, state.OnStart(world))
	assert.NotNil(t, state.mount)
	assert.NotNil(t, state.windowMount)
	assert.NotNil(t, state.equipMount)
	assert.Equal(t, charSubBrowse, state.subState, "初期状態は閲覧")
}

func TestCharacterState_装備スロットは武器5と防具7の合計12(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	props := state.fetchProps(world)
	assert.Len(t, props.EquipSlots, 12, "武器5スロットと防具7スロット")
}

func TestCharacterState_情報タブは能力スキル効果健康基本の5つ(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	props := state.fetchProps(world)
	labels := make([]string, len(props.InfoTabs))
	for i, tab := range props.InfoTabs {
		labels[i] = tab.Label
	}
	assert.Equal(t, []string{"能力", "スキル", "効果", "健康", "基本"}, labels)
}

func TestCharacterState_スキルタブはカテゴリ見出しを含む(t *testing.T) {
	t.Parallel()

	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	props := state.fetchProps(world)
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

func TestNextPolicy_端で循環し未知の値は先頭を返す(t *testing.T) {
	t.Parallel()
	all := []string{"a", "b", "c"}
	assert.Equal(t, "b", nextPolicy(all, "a"))
	assert.Equal(t, "c", nextPolicy(all, "b"))
	assert.Equal(t, "a", nextPolicy(all, "c"), "末尾の次は先頭へ循環する")
	assert.Equal(t, "a", nextPolicy(all, "x"), "未知の値は先頭を返す")
}

func TestFetchCommandRows_仲間はポリシーと解雇を持ちプレイヤーは持たない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)

	rows := fetchCommandRows(world, member)
	require.Len(t, rows, 6, "位置・戦闘・回収・処理・補給・解雇の6行")
	assert.Equal(t, cmdMovement, rows[0].Kind)
	assert.Equal(t, cmdDismiss, rows[5].Kind, "末尾は解雇")

	assert.Nil(t, fetchCommandRows(world, player), "SquadAI を持たないプレイヤーは命令行を持たない")
}

func TestDetailPageCount_componentが多いレイガンは複数ページになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	entity, err := lifecycle.SpawnBackpackItem(world, "レイガン", 1)
	require.NoError(t, err)
	assert.Greater(t, menuscreen.DetailPageCount(world, entity), 1, "性能区画が多いアイテムの詳細は複数ページに分割される")
}

func TestCharacterState_cycleCommandは位置ポリシーを次の値へ進める(t *testing.T) {
	t.Parallel()
	state := &CharacterState{}
	world := testutil.InitTestWorld(t)
	require.NoError(t, state.OnStart(world))
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)

	state.target = member
	before := query.GetSquadAI(world, member).Movement
	state.cycleCommand(world, cmdMovement)
	after := query.GetSquadAI(world, member).Movement
	assert.Equal(t, nextPolicy(gc.AllSquadMovements(), before), after, "位置ポリシーは次の値へ進む")
}
