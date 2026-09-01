package overworld

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countStoredItems は収納に入っているアイテムの数を返す。LocationInStorage を持つエンティティを数える。
func countStoredItems(world w.World) int {
	q := ecs.NewFilter1[gc.LocationInStorage](world.ECS).Query()
	n := 0
	for q.Next() {
		n++
	}
	return n
}

// countFieldItems は床に置かれたアイテムの数を返す。LocationOnField を持つエンティティを数える。
func countFieldItems(world w.World) int {
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	n := 0
	for q.Next() {
		n++
	}
	return n
}

// TestSpawnFieldLoot_床に戦利品が出る は床 loot の配線を固定する。loot の item group を引き、抽選したアイテムを
// LocationOnField として床へ置く。KindLoot placement を実アイテムへ実体化する経路を守る。
func TestSpawnFieldLoot_床に戦利品が出る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))

	// healing_item は distribution なので必ず1件を返す。床に LocationOnField のアイテムが増える
	pos := consts.Coord[consts.Tile]{X: 0, Y: 0}
	require.NoError(t, spawnFieldLoot(world, "healing_item", pos, rng))
	assert.Positive(t, countFieldItems(world), "loot group から床にアイテムが1つ以上出る")
}

// TestSpawnFieldLoot_未存在グループはエラー は写像先の typo や raw 側削除を spawn 時に検知することを守る。
func TestSpawnFieldLoot_未存在グループはエラー(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))
	pos := consts.Coord[consts.Tile]{X: 0, Y: 0}
	require.Error(t, spawnFieldLoot(world, "no_such_group", pos, rng))
}

// TestPopulateStorageLoot_収納家具に戦利品が入る は収納 loot の配線を固定する。収納 prop(押入れ)には ruins の
// item テーブルから戦利品が入り、収納でない prop(蝋燭)には入らない。家具別の loot を ruins 既存の
// DropTable/ItemTable 機構で実現していることを守る。
func TestPopulateStorageLoot_収納家具に戦利品が入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))

	closet, err := lifecycle.SpawnProp(world, "closet", 0, 0)
	require.NoError(t, err)
	require.NoError(t, populateStorageLoot(world, closet, "closet", rng))
	assert.Positive(t, countStoredItems(world), "収納家具の押入れに戦利品が1つ以上入る")

	// 家具別テーブルの分化。冷蔵庫は食料庫テーブルから引く。テーブル解決と loot 産出を確かめる
	beforeFridge := countStoredItems(world)
	fridge, err := lifecycle.SpawnProp(world, "refrigerator", 2, 2)
	require.NoError(t, err)
	require.NoError(t, populateStorageLoot(world, fridge, "refrigerator", rng))
	assert.Greater(t, countStoredItems(world), beforeFridge, "冷蔵庫に食料庫テーブルの戦利品が入る")

	before := countStoredItems(world)
	candle, err := lifecycle.SpawnProp(world, "candle", 1, 1)
	require.NoError(t, err)
	require.NoError(t, populateStorageLoot(world, candle, "candle", rng))
	assert.Equal(t, before, countStoredItems(world), "収納でない蝋燭は戦利品を入れない")
}

// TestInteriorPropRaw_全施設の家具refが写像を持つ は、施設 content が生む家具が in-game で無言に欠落
// しないことを守る。各施設種別を Furnish し、KindFurniture の Ref がすべて 写像 PropRawName にあることを
// 確かめる。content へ家具を足して写像を忘れると、ここで落ちて気付ける。
func TestInteriorPropRaw_全施設の家具refが写像を持つ(t *testing.T) {
	t.Parallel()

	// 単室 Furnish と多部屋 FurnishBuilding の両経路をなめる。多部屋は民家の水回りなど別の家具を出すので
	// 両方を検査しないと写像漏れを見逃す
	small := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	big := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 10, Y: 13}
	bigDoor := interior.Vec{X: 14, Y: 0}
	check := func(fac interior.FacilityKind, placed []interior.Placed) {
		for _, p := range placed {
			if p.Kind != interior.KindFurniture {
				continue
			}
			_, ok := interior.PropRawName(p.Ref)
			assert.Truef(t, ok, "施設 %q の家具 %q は 写像を持つ", fac, p.Ref)
		}
	}
	for _, fac := range []interior.FacilityKind{"house", "store", "clinic", "office", "depot", "antique", "lab", ""} {
		check(fac, interior.Furnish(1, small, door, fac))
		_, placed := interior.FurnishBuilding(1, big, bigDoor, fac)
		check(fac, placed)
	}
}

// TestInteriorPropRaw_写像先のrawが実在する は、抽象 Ref の写像先がゲームの raw prop に存在することを
// 守る。raw 名の typo や raw 側の削除で spawn 時にエラーになる退行を、生成を待たず表の検査で捕まえる。
func TestInteriorPropRaw_写像先のrawが実在する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	for ref, name := range interior.PropRaws() {
		_, err := raw.GetProp(world.Resources.RawMaster, name)
		require.NoErrorf(t, err, "Ref %q の写像先 raw %q が実在する", ref, name)
	}
}

// TestInteriorLootRaw_全施設のloot_refが写像を持つ は、施設 content が生む床 loot が in-game で無言に欠落
// しないことを守る。各施設を Furnish し、KindLoot の Ref がすべて 写像 LootGroupName にあることを確かめる。
// content へ loot を足して lootRaw への写像を忘れると、ここで落ちて気付ける。KindFurniture の写像検査と対称。
func TestInteriorLootRaw_全施設のloot_refが写像を持つ(t *testing.T) {
	t.Parallel()

	small := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	big := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 10, Y: 13}
	bigDoor := interior.Vec{X: 14, Y: 0}
	sawLoot := false
	check := func(fac interior.FacilityKind, placed []interior.Placed) {
		for _, p := range placed {
			if p.Kind != interior.KindLoot {
				continue
			}
			sawLoot = true
			_, ok := interior.LootGroupName(p.Ref)
			assert.Truef(t, ok, "施設 %q の loot %q は写像を持つ", fac, p.Ref)
		}
	}
	for _, fac := range []interior.FacilityKind{"house", "store", "clinic", "office", "depot", "antique", "lab", ""} {
		check(fac, interior.Furnish(1, small, door, fac))
		_, placed := interior.FurnishBuilding(1, big, bigDoor, fac)
		check(fac, placed)
	}
	assert.True(t, sawLoot, "少なくとも1施設が KindLoot を生む。床 loot のレールが有効であることを固定する")
}

// TestInteriorLootRaw_写像先のitem_groupが実在する は、loot Ref の写像先がゲームの raw item group に存在する
// ことを守る。group 名の typo や raw 側の削除で spawn 時にエラーになる退行を、生成を待たず表の検査で捕まえる。
func TestInteriorLootRaw_写像先のitem_groupが実在する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	for ref, group := range interior.LootGroups() {
		_, err := raw.GetItemGroup(world.Resources.RawMaster, group)
		require.NoErrorf(t, err, "loot Ref %q の写像先 item group %q が実在する", ref, group)
	}
}

// TestFurnishBuilding_屋内床にShelterが焼かれる は、建物内部の床タイルに屋内 Shelter が焼かれ、庭や壁には
// 焼かれないことを固定する。オーバーワールド建物を屋内と判定し、温度式が per-tile の屋内緩和を効かせる配線を守る。
func TestFurnishBuilding_屋内床にShelterが焼かれる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: 50, chunkH: 50, tiles: &tileIndex{world: world, loX: 0, hiX: 50}}
	footprint := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	door := interior.Vec{X: 10, Y: 13}
	_, _, err := furnishBuilding(world, g, footprint, door, facilityHouse, 1)
	require.NoError(t, err)

	floorTotal, floorFull, nonFloorFull := 0, 0, 0
	for _, e := range g.tiles.get() {
		if !world.Components.TileTemperature.Has(e) {
			continue
		}
		shelter := world.Components.TileTemperature.Get(e).Shelter
		switch world.Components.RawID.Get(e).ID {
		case consts.TileNameFloor:
			floorTotal++
			if shelter == gc.ShelterFull {
				floorFull++
			}
		case consts.TileNameDirt, consts.TileNameDWall:
			if shelter == gc.ShelterFull {
				nonFloorFull++
			}
		}
	}
	require.Positive(t, floorTotal, "建物内部に床タイルがある")
	assert.Equal(t, floorTotal, floorFull, "屋内の床タイルはすべて ShelterFull を持つ")
	assert.Zero(t, nonFloorFull, "庭の土と壁には屋内 Shelter を焼かない")
}
