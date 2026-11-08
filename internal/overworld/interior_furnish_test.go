package overworld

import (
	"math/rand/v2"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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

// TestPopulateStorageLoot_収納家具に戦利品が入る は収納 loot の配線を固定する。収納 prop(押入れ)には渡した
// item テーブルから戦利品が入り、収納でない prop(蝋燭)には入らない。家は荒らされ率を持つので複数置いて数える。
func TestPopulateStorageLoot_収納家具に戦利品が入る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	rng := rand.New(rand.NewPCG(1, 0x4))

	for i := range 10 {
		closet, err := lifecycle.SpawnProp(world, "closet", consts.Tile(i), 0)
		require.NoError(t, err)
		require.NoError(t, populateStorageLoot(world, closet, "closet", "house_closet", "house", rng))
	}
	assert.Positive(t, countStoredItems(world), "収納家具の押入れに戦利品が入る")

	before := countStoredItems(world)
	candle, err := lifecycle.SpawnProp(world, "candle", 1, 1)
	require.NoError(t, err)
	require.NoError(t, populateStorageLoot(world, candle, "candle", "", "house", rng))
	assert.Equal(t, before, countStoredItems(world), "収納でない蝋燭は戦利品を入れない")
}

func TestPopulateStorageLoot_未登録の施設はエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	closet, err := lifecycle.SpawnProp(world, "closet", 0, 0)
	require.NoError(t, err)
	require.Error(t, populateStorageLoot(world, closet, "closet", "house_closet", "no_such", rand.New(rand.NewPCG(1, 0x4))))
}

func TestPopulateStorageLoot_表の無い収納家具はエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	closet, err := lifecycle.SpawnProp(world, "closet", 0, 0)
	require.NoError(t, err)
	require.Error(t, populateStorageLoot(world, closet, "closet", "", "house", rand.New(rand.NewPCG(1, 0x4))))
}

func TestPopulateStorageLoot_施設の荒らされ率で空のまま残す(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		fac       string
		wantEmpty int
	}{
		{"店は8割が空", "store", 80},
		{"骨董品店は5割が空", "antique", 50},
		{"倉庫は4割が空", "depot", 40},
		{"家は3割が空", "house", 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			rng := rand.New(rand.NewPCG(1, 0x4))
			const n = 400
			empty := 0
			for i := range n {
				ent, err := lifecycle.SpawnProp(world, "goods_shelf", consts.Tile(i), 0)
				require.NoError(t, err)
				require.NoError(t, populateStorageLoot(world, ent, "goods_shelf", "store_goods", tt.fac, rng))
				if len(query.GetStorageItems(world, ent)) == 0 {
					empty++
				}
			}
			assert.InDelta(t, tt.wantEmpty, empty*100/n, 6, "空の割合が施設の荒らされ率に近い")
		})
	}
}

func TestFurnishBuilding_施設の部屋の定義が書いた表だけを引く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	raws := world.Resources.RawMaster
	tests := []struct {
		fac  string
		want []string
	}{
		{"house", []string{"house_closet", "house_kitchen"}},
		{"store", []string{"store_goods", "pantry", "medicine_shelf", "office_storage"}},
		{"antique", []string{"antique_shelf", "office_storage"}},
		{"depot", []string{"depot_storage"}},
	}
	for _, tt := range tests {
		used := map[string]bool{}
		for seed := range uint64(30) {
			_, placed, err := interior.FurnishBuilding(raws, seed, interior.Rect{X: 0, Y: 0, W: 28, H: 20}, interior.Vec{X: 14, Y: 0}, tt.fac)
			require.NoError(t, err)
			for _, p := range placed {
				if p.LootTable != "" {
					used[p.LootTable] = true
				}
			}
		}
		got := make([]string, 0, len(used))
		for id := range used {
			got = append(got, id)
		}
		assert.ElementsMatch(t, tt.want, got, "施設 %s", tt.fac)
	}
}

// TestInteriorContents_収納家具は戦利品表を書く は、部屋の定義の全ての家具と衛星について、収納家具なら
// lootTable を書いていることを確かめる。抽選に依らず定義を全て舐めるので、出にくい変種の書き漏れも落ちる。
func TestInteriorContents_収納家具は戦利品表を書く(t *testing.T) {
	t.Parallel()
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	isStorage := func(ref string) bool {
		name, ok := interior.PropRawName(ref)
		if !ok {
			return false
		}
		pr, err := raw.GetProp(raws, name)
		return err == nil && pr.Storage != nil
	}
	for _, ic := range raw.PtrSlice(raws.InteriorContents) {
		for _, g := range raw.PtrSlice(ic.Groups) {
			for _, it := range g.Items {
				if it.Kind == oapi.Furniture && isStorage(it.Ref) {
					assert.NotNilf(t, it.LootTable, "部屋の定義 %q の収納家具 %q は戦利品表を書く", ic.Id, it.Ref)
				}
				for _, sat := range raw.PtrSlice(it.Satellites) {
					if sat.Kind == oapi.Furniture && isStorage(sat.Ref) {
						assert.NotNilf(t, sat.LootTable, "部屋の定義 %q の収納家具 %q の衛星は戦利品表を書く", ic.Id, sat.Ref)
					}
				}
			}
		}
	}
}

// TestInteriorPropRaw_全施設の家具と敵のrefが写像を持つ は、施設 content が生む家具と敵が in-game で無言に欠落
// しないことを守る。各施設種別を Furnish し、家具の Ref が PropRawName に、敵の Ref が IsBeingRef にあることを
// 確かめる。content へ足して写像を忘れると、ここで落ちて気付ける。
func TestInteriorPropRaw_全施設の家具と敵のrefが写像を持つ(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	raws := world.Resources.RawMaster

	// 単室 Furnish と多部屋 FurnishBuilding の両経路をなめる。多部屋は民家の水回りなど別の家具を出すので
	// 両方を検査しないと写像漏れを見逃す
	small := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	big := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 10, Y: 13}
	bigDoor := interior.Vec{X: 14, Y: 0}
	check := func(fac string, placed []interior.Placed) {
		for _, p := range placed {
			switch p.Kind {
			case interior.KindFurniture:
				_, ok := interior.PropRawName(p.Ref)
				assert.Truef(t, ok, "施設 %q の家具 %q は 写像を持つ", fac, p.Ref)
			case interior.KindBeing:
				assert.Truef(t, interior.IsBeingRef(p.Ref), "施設 %q の敵 %q は spawn される Ref", fac, p.Ref)
			case interior.KindLoot, interior.KindDecor, interior.KindTrap:
			}
		}
	}
	for _, f := range raw.PtrSlice(raws.Facilities) {
		fac := f.Id
		single, err := interior.Furnish(raws, 1, small, door, fac)
		require.NoError(t, err)
		check(fac, single)
		_, placed, err := interior.FurnishBuilding(raws, 1, big, bigDoor, fac)
		require.NoError(t, err)
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

	world := testutil.InitTestWorld(t)
	raws := world.Resources.RawMaster

	small := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	big := interior.Rect{X: 0, Y: 0, W: 28, H: 20}
	door := interior.Vec{X: 10, Y: 13}
	bigDoor := interior.Vec{X: 14, Y: 0}
	sawLoot := false
	check := func(fac string, placed []interior.Placed) {
		for _, p := range placed {
			if p.Kind != interior.KindLoot {
				continue
			}
			sawLoot = true
			_, ok := interior.LootGroupName(p.Ref)
			assert.Truef(t, ok, "施設 %q の loot %q は写像を持つ", fac, p.Ref)
		}
	}
	for _, f := range raw.PtrSlice(raws.Facilities) {
		fac := f.Id
		single, err := interior.Furnish(raws, 1, small, door, fac)
		require.NoError(t, err)
		check(fac, single)
		_, placed, err := interior.FurnishBuilding(raws, 1, big, bigDoor, fac)
		require.NoError(t, err)
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

func TestSpawnInteriorBeing(t *testing.T) {
	t.Parallel()

	t.Run("施設の敵の表から1体を指定の位置へ置く", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		table, err := urbanEnemyTableFor(world.Resources.RawMaster, "store")
		require.NoError(t, err)
		members := map[string]bool{}
		for _, e := range table.Entries {
			members[e.Id] = true
		}
		pos := consts.Coord[consts.Tile]{X: 3, Y: 4}
		require.NoError(t, spawnInteriorBeing(world, "store", 1, pos, rand.New(rand.NewPCG(1, 0x5))))

		var got []string
		q := ecs.NewFilter2[gc.FactionEnemy, gc.GridElement](world.ECS).Query()
		for q.Next() {
			_, ge := q.Get()
			assert.Equal(t, pos, ge.Coord)
			got = append(got, query.GetEntityID(q.Entity(), world))
		}
		require.Len(t, got, 1)
		assert.True(t, members[got[0]], "店の敵の表から引く")
	})

	t.Run("未登録の施設はエラー", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		require.Error(t, spawnInteriorBeing(world, "no_such", 1, consts.Coord[consts.Tile]{X: 3, Y: 4}, rand.New(rand.NewPCG(1, 0x5))))
	})
}

func TestFurnishBuilding_部屋の定義に従い建物の中に敵を置く(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: 50, chunkH: 50, tiles: &tileIndex{world: world, loX: 0, hiX: 50}}
	footprint := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	door := interior.Vec{X: 10, Y: 13}
	_, _, err := furnishBuilding(world, g, footprint, door, "store", 1, 1)
	require.NoError(t, err)

	table, err := urbanEnemyTableFor(world.Resources.RawMaster, "store")
	require.NoError(t, err)
	members := map[string]bool{}
	for _, e := range table.Entries {
		members[e.Id] = true
	}
	inside := 0
	q := ecs.NewFilter2[gc.FactionEnemy, gc.GridElement](world.ECS).Query()
	for q.Next() {
		_, ge := q.Get()
		assert.True(t, ge.X < footprint.W && ge.Y < footprint.H, "敵は建物の区画の中にいる")
		assert.True(t, members[query.GetEntityID(q.Entity(), world)], "敵は店の敵の表から引く")
		inside++
	}
	assert.Positive(t, inside)
}

// TestFurnishBuilding_屋内床にShelterが設定される は、建物内部の床タイルに屋内 Shelter が書き込まれ、庭や壁には
// 書き込まれないことを固定する。オーバーワールド建物を屋内と判定し、温度式が per-tile の屋内緩和を効かせる配線を守る。
func TestFurnishBuilding_屋内床にShelterが設定される(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	g := chunkGeom{offsetX: 0, offsetY: 0, chunkW: 50, chunkH: 50, tiles: &tileIndex{world: world, loX: 0, hiX: 50}}
	footprint := interior.Rect{X: 0, Y: 0, W: 20, H: 14}
	door := interior.Vec{X: 10, Y: 13}
	_, _, err := furnishBuilding(world, g, footprint, door, "house", 1, 1)
	require.NoError(t, err)

	floorTotal, floorFull, nonFloorFull := 0, 0, 0
	for _, e := range g.tiles.get() {
		if !world.Components.TileEnvironment.Has(e) {
			continue
		}
		shelter := world.Components.TileEnvironment.Get(e).Shelter
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
	assert.Zero(t, nonFloorFull, "庭の土と壁には屋内 Shelter を書き込まない")
}
