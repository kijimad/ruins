package lifecycle_test

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpawnCube_運転できる移動拠点として生成される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 7, Y: 8})
	require.NoError(t, err)

	assert.True(t, world.Components.Drivable.Has(cube), "運転できる印を持つ")
	assert.False(t, world.Components.BlockPass.Has(cube), "直上に立てるよう通行可能にする")
	assert.True(t, world.Components.Prop.Has(cube), "設置物である")
	require.True(t, world.Components.HP.Has(cube), "prop 標準の耐久度を持つ")
	assert.Positive(t, world.Components.HP.Get(cube).Max, "HP 上限は正")
	assert.Equal(t, world.Components.HP.Get(cube).Max, world.Components.HP.Get(cube).Current, "生成直後は満タン")
	assert.True(t, world.Components.WeightCapacity.Has(cube), "収納容量を持つ")
	require.True(t, world.Components.Interactable.Has(cube), "相互作用を持つ")
	assert.Contains(t, world.Components.Interactable.Get(cube).Interactions, gc.InteractionOpenCubeMenu, "直上でキューブメニューを開ける")
	require.True(t, world.Components.GridElement.Has(cube))
	assert.Equal(t, consts.Coord[consts.Tile]{X: 7, Y: 8}, world.Components.GridElement.Get(cube).Coord)
	require.True(t, world.Components.StageBound.Has(cube), "帯へ束縛される")
	assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(cube).Key)
}

// TestSpawnDungeonEntrance_ダンジョンポータルと同じアニメフレームを持つ は、オーバーワールドの
// 遺跡入口がダンジョン内の階段ポータルと同じ回転アニメを持つことを固定する。入口はコードで
// 組むため、以前はアニメフレーム AnimKeys が抜けて静止していた。raw の warp_next を流用して揃える。
func TestSpawnDungeonEntrance_ダンジョンポータルと同じアニメフレームを持つ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	e, err := lifecycle.SpawnDungeonEntrance(world, 5, 5, "亡者の森")
	require.NoError(t, err)

	require.True(t, world.Components.SpriteRender.Has(e), "スプライトを持つ")
	assert.NotEmpty(t, world.Components.SpriteRender.Get(e).AnimKeys, "入口はアニメフレームを持ちアニメーションする")

	// 相互作用は遺跡進入で、warp_next 由来の次階ポータルではない
	require.True(t, world.Components.Interactable.Has(e), "相互作用を持つ")
	assert.Contains(t, world.Components.Interactable.Get(e).Interactions, gc.InteractionDungeonEnter, "遺跡進入の相互作用")

	// オーバーワールドの地物として帯へ束縛される
	require.True(t, world.Components.StageBound.Has(e), "ステージへ束縛される")
	assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(e).Key, "オーバーワールド帯へ束縛される")

	// 遺跡定義名を運ぶ
	require.True(t, world.Components.DungeonEntrance.Has(e), "遺跡入口コンポーネントを持つ")
	assert.Equal(t, "亡者の森", world.Components.DungeonEntrance.Get(e).DefinitionName, "進入先の遺跡定義名を運ぶ")
}

func TestOpenDoor_開くと通行と視界を遮らなくなる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 3, Y: 4}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	require.True(t, world.Components.BlockPass.Has(door), "生成直後は通行不可")
	require.True(t, world.Components.BlockView.Has(door), "生成直後は視界を遮る")

	require.NoError(t, lifecycle.OpenDoor(world, door))

	assert.True(t, world.Components.Door.Get(door).IsOpen, "開いた状態になる")
	assert.False(t, world.Components.BlockPass.Has(door), "通行可能になる")
	assert.False(t, world.Components.BlockView.Has(door), "視界を遮らなくなる")
	assert.Equal(t, "door_horizontal_open", world.Components.SpriteRender.Get(door).SpriteKey)
}

func TestOpenDoor_縦向きの扉は縦向きのスプライトになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 3, Y: 4}, gc.DoorOrientationVertical)
	require.NoError(t, err)

	require.NoError(t, lifecycle.OpenDoor(world, door))

	assert.Equal(t, "door_vertical_open", world.Components.SpriteRender.Get(door).SpriteKey)
}

func TestOpenDoor_扉でないエンティティはエラーになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	notDoor := world.ECS.NewEntity()

	err := lifecycle.OpenDoor(world, notDoor)

	require.Error(t, err)
	assert.EqualError(t, err, "entity is not a door")
}

func TestCloseDoor_閉じると通行不可と視界遮断に戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 3, Y: 4}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	require.NoError(t, lifecycle.OpenDoor(world, door))

	require.NoError(t, lifecycle.CloseDoor(world, door))

	assert.False(t, world.Components.Door.Get(door).IsOpen, "閉じた状態になる")
	assert.True(t, world.Components.BlockPass.Has(door), "通行不可に戻る")
	assert.True(t, world.Components.BlockView.Has(door), "視界を遮るようになる")
	assert.Equal(t, "door_horizontal_closed", world.Components.SpriteRender.Get(door).SpriteKey)
}

func TestCloseDoor_扉でないエンティティはエラーになる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	notDoor := world.ECS.NewEntity()

	err := lifecycle.CloseDoor(world, notDoor)

	require.Error(t, err)
	assert.EqualError(t, err, "entity is not a door")
}

func TestSpawnDoor_扉は殴って壊せる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 3, Y: 4}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)

	require.True(t, world.Components.HP.Has(door), "扉は HP を持ち破壊できる")
	assert.Positive(t, world.Components.HP.Get(door).Max)
	require.True(t, world.Components.Interactable.Has(door))
	assert.Contains(t, world.Components.Interactable.Get(door).Interactions, gc.InteractionMelee, "扉は殴る対象になる")
}

func TestSpawnDoor_扉のHPはrawのdoor定義と一致する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 3, Y: 4}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)

	// bespoke な SpawnDoor の doorHP が raw の door prop から乖離したら失敗する。
	// raw と bespoke の二重管理を人手のコメントでなくテストで担保する
	rawSpec, err := raw.NewPropSpec(world.Resources.RawMaster, "door")
	require.NoError(t, err)
	require.NotNil(t, rawSpec.HP)
	assert.Equal(t, rawSpec.HP.Max, world.Components.HP.Get(door).Max, "SpawnDoor の扉 HP は raw の door 定義と一致すべき")
}
