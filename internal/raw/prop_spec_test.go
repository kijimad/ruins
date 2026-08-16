package raw

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPropSpec_視界を遮るPropにはBlockViewが設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "壁"
id = "壁"
Description = "視界を遮る壁"
BlockPass = false
BlockView = true

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "wall"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "壁")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.BlockView, "BlockView=trueのPropにはBlockViewコンポーネントが設定されるべき")
}

func TestNewPropSpec_通行不可を伴わない通行コストが設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "泥濘"
id = "泥濘"
Description = "移動に時間がかかる地面"
BlockPass = false
BlockView = false
PassCost = 50

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "mud"
Depth = 1
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "泥濘")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.PassCost)
	assert.Equal(t, 50, entitySpec.PassCost.Value)
}

func TestNewPropSpec_扉が設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "木の扉"
id = "木の扉"
Description = "開閉できる扉"
BlockPass = false
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "door"
Depth = 1

[Props.Door]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "木の扉")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.Door)
	assert.False(t, entitySpec.Door.IsOpen)
	assert.Equal(t, gc.DoorOrientationHorizontal, entitySpec.Door.Orientation)
	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionDoor)
}

func TestNewPropSpec_次階層ワープトリガーが設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "下り階段"
id = "下り階段"
Description = "1つ下の階層へ進むポータル"
BlockPass = false
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "warp_next"
Depth = 1

[Props.WarpNextTrigger]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "下り階段")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionPortalNext)
}

func TestNewPropSpec_キューブ退場トリガーが設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "キューブ出口"
id = "キューブ出口"
Description = "移動拠点キューブから退場するポータル"
BlockPass = false
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "cube_exit"
Depth = 1

[Props.WarpCubeExitTrigger]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "キューブ出口")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionExitCube)
}

func TestNewPropSpec_キューブコントロールパネルが設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "操作パネル"
id = "操作パネル"
Description = "移動拠点キューブのコントロールパネル"
BlockPass = false
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "cube_panel"
Depth = 1

[Props.CubePanelTrigger]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "操作パネル")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionCubePanel)
}

func TestNewPropSpec_出荷場所が設定される(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "出荷場所"
id = "出荷場所"
Description = "通信販売の出荷場所"
BlockPass = true
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "shipping_station"
Depth = 1

[Props.ShippingStation]
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	entitySpec, err := NewPropSpec(raws, "出荷場所")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.AuctionStation)
	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionAuction)
}

func TestNewPropSpec_収納の重量表記が不正だとエラー(t *testing.T) {
	t.Parallel()
	str := `
[[Props]]
Name = "壊れた収納"
id = "壊れた収納"
Description = "重量表記が壊れている"
BlockPass = false
BlockView = false

[Props.SpriteRender]
SpriteSheetName = "field"
SpriteKey = "broken_storage"
Depth = 1

[Props.Storage]
MaxWeight = "abc"
`
	raws, err := DecodeRaws(str)
	require.NoError(t, err)

	_, err = NewPropSpec(raws, "壊れた収納")
	require.Error(t, err)
	assert.ErrorContains(t, err, "壊れた収納' max weight")
}

func TestNewPropSpec_分解定義があるとInteractionDisassembleが設定される(t *testing.T) {
	t.Parallel()
	raws := oapi.Raws{
		Props: &[]oapi.Prop{
			{
				Id:   "解体台",
				Name: "解体台",
				SpriteRender: oapi.SpriteRender{
					SpriteSheetName: "field",
					SpriteKey:       "workbench",
				},
				Disassembly: &oapi.Disassembly{
					ToolCategory: oapi.Prying,
					BaseAP:       100,
					Yields:       []oapi.DisassemblyYield{{Id: "鉄くず", Count: "1d1"}},
				},
			},
		},
	}

	entitySpec, err := NewPropSpec(raws, "解体台")
	require.NoError(t, err)

	require.NotNil(t, entitySpec.Interactable)
	assert.Contains(t, entitySpec.Interactable.Interactions, gc.InteractionDisassemble)
}

func TestNewPropSpec_未存在のPropはKeyNotFoundErrorになる(t *testing.T) {
	t.Parallel()
	raws, err := DecodeRaws("")
	require.NoError(t, err)

	_, err = NewPropSpec(raws, "存在しないProp")
	require.Error(t, err)

	var keyErr KeyNotFoundError
	require.ErrorAs(t, err, &keyErr)
	assert.Equal(t, "存在しないProp", keyErr.Key)
	assert.Equal(t, "Props", keyErr.Collection)
}
