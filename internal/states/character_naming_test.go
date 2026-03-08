package states

import (
	"testing"
	"unicode/utf8"

	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/hooks"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNameValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "1文字の名前は有効",
			input:    "A",
			expected: true,
		},
		{
			name:     "10文字の名前は有効",
			input:    "ABCDEFGHIJ",
			expected: true,
		},
		{
			name:     "空文字は無効",
			input:    "",
			expected: false,
		},
		{
			name:     "11文字の名前は無効",
			input:    "ABCDEFGHIJK",
			expected: false,
		},
		{
			name:     "日本語1文字は有効",
			input:    "あ",
			expected: true,
		},
		{
			name:     "日本語10文字は有効",
			input:    "あいうえおかきくけこ",
			expected: true,
		},
		{
			name:     "日本語11文字は無効",
			input:    "あいうえおかきくけこさ",
			expected: false,
		},
		{
			name:     "混合文字で10文字は有効",
			input:    "Ash太郎1234",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			nameLen := utf8.RuneCountInString(tt.input)
			isValid := nameLen >= nameMinLength && nameLen <= nameMaxLength
			assert.Equal(t, tt.expected, isValid)
		})
	}
}

func TestCharacterNamingState_OnStart(t *testing.T) {
	t.Parallel()

	state := &CharacterNamingState{}
	world := testutil.InitTestWorld(t)

	err := state.OnStart(world)
	require.NoError(t, err)
	assert.NotNil(t, state.mount, "mountが初期化されている")
}

func TestCharacterNamingState_OnStart_WithExistingPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	state := &CharacterNamingState{}
	require.NoError(t, state.OnStart(world))

	props := state.mount.GetProps()
	assert.Equal(t, "Ash", props.CurrentName, "既存プレイヤー名が初期値に設定される")
}

func TestConfirmName_ChangesPlayerName(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.mount = hooks.NewMount[namingProps]()
	st.mount.SetProps(namingProps{CurrentName: "NewName"})

	transition := st.confirmName(world)
	assert.Equal(t, es.TransPop, transition.Type, "名前変更成功でTransPop")

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "NewName", nameComp.Name)
}

func TestConfirmName_Japanese(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.mount = hooks.NewMount[namingProps]()
	st.mount.SetProps(namingProps{CurrentName: "太郎"})

	transition := st.confirmName(world)
	assert.Equal(t, es.TransPop, transition.Type, "日本語名で成功")

	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "太郎", nameComp.Name)
}

func TestConfirmName_InvalidLength(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.mount = hooks.NewMount[namingProps]()

	// 空文字は無効
	st.mount.SetProps(namingProps{CurrentName: ""})
	transition := st.confirmName(world)
	assert.Equal(t, es.TransNone, transition.Type, "空文字でTransNone")

	props := st.mount.GetProps()
	assert.NotEmpty(t, props.ErrorMessage, "エラーメッセージが設定される")

	// 名前は変更されていない
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "Ash", nameComp.Name)
}

func TestConfirmName_TooLong(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	st := &CharacterNamingState{}
	st.mount = hooks.NewMount[namingProps]()

	// 11文字は無効
	st.mount.SetProps(namingProps{CurrentName: "ABCDEFGHIJK"})
	transition := st.confirmName(world)
	assert.Equal(t, es.TransNone, transition.Type, "長すぎる名前でTransNone")

	props := st.mount.GetProps()
	assert.NotEmpty(t, props.ErrorMessage, "エラーメッセージが設定される")

	// 名前は変更されていない
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	require.NoError(t, err)
	nameComp := world.Components.Name.Get(playerEntity).(*gc.Name)
	assert.Equal(t, "Ash", nameComp.Name)
}

func TestConfirmName_NewPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// プレイヤーがいない状態

	st := &CharacterNamingState{}
	st.mount = hooks.NewMount[namingProps]()
	st.mount.SetProps(namingProps{CurrentName: "NewPlayer"})

	transition := st.confirmName(world)
	assert.Equal(t, es.TransPush, transition.Type, "新規プレイヤーでTransPush")
}

func TestCharacterNamingState_DoAction_Cancel(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := worldhelper.SpawnPlayer(world, 5, 5, "Ash")
	require.NoError(t, err)

	state := &CharacterNamingState{}
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransPop, transition.Type, "既存プレイヤーでキャンセルはTransPop")
}

func TestCharacterNamingState_DoAction_Cancel_NewPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	// プレイヤーがいない状態

	state := &CharacterNamingState{}
	require.NoError(t, state.OnStart(world))

	transition, err := state.DoAction(world, inputmapper.ActionMenuCancel)
	require.NoError(t, err)
	assert.Equal(t, es.TransReplace, transition.Type, "新規プレイヤーでキャンセルはTransReplace")
}

func TestNewCharacterNamingState(t *testing.T) {
	t.Parallel()

	factory := NewCharacterNamingState
	state := factory()

	assert.NotNil(t, state, "Stateが作成される")
	_, ok := state.(*CharacterNamingState)
	assert.True(t, ok, "CharacterNamingState型である")
}

func TestCharacterNamingState_String(t *testing.T) {
	t.Parallel()

	state := &CharacterNamingState{}
	assert.Equal(t, "CharacterNaming", state.String())
}
