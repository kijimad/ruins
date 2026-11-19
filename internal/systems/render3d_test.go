package systems

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	"github.com/kijimaD/ruins/internal/render3d"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSampleOf はタイル描画情報から明るさと色のサンプルを起こす分岐を固定する。
// 3Dの Draw を通さず、Darkness の明るさ反映と可視/記憶/未探索の分岐を単体で検証する。
func TestSampleOf(t *testing.T) {
	t.Parallel()

	t.Run("可視はDarknessを明るさへ反映する", func(t *testing.T) {
		t.Parallel()
		s, drawable, visible := sampleOf(TileRenderVisible{Darkness: 0.3})
		assert.InDelta(t, 0.7, s.brightness, 1e-9, "brightness = 1 - Darkness")
		assert.True(t, drawable)
		assert.True(t, visible)
		assert.Equal(t, [3]float64{1, 1, 1}, s.color, "無色の光源は白のまま")
	})

	t.Run("可視の光源色は最大成分で正規化して色味だけ返す", func(t *testing.T) {
		t.Parallel()
		s, _, _ := sampleOf(TileRenderVisible{LightColor: color.RGBA{R: 255, G: 128, B: 0, A: 255}})
		assert.InDelta(t, 1.0, s.color[0], 1e-9)
		assert.InDelta(t, 128.0/255, s.color[1], 1e-9)
		assert.InDelta(t, 0.0, s.color[2], 1e-9)
	})

	t.Run("記憶は描画可だが可視ではない", func(t *testing.T) {
		t.Parallel()
		s, drawable, visible := sampleOf(TileRenderRemembered{Darkness: 0.75})
		assert.InDelta(t, 0.25, s.brightness, 1e-9)
		assert.True(t, drawable)
		assert.False(t, visible)
		assert.Equal(t, [3]float64{1, 1, 1}, s.color, "記憶は光源色を持たず白")
	})

	t.Run("未探索は描画しない", func(t *testing.T) {
		t.Parallel()
		_, drawable, visible := sampleOf(nil)
		assert.False(t, drawable)
		assert.False(t, visible)
	})
}

// TestLightSample_overbrightとtint はフィルタ列の合成を固定する。
// 明るさを持ち上げてから色に掛け、乗算色を作ることを検証する。
func TestLightSample_overbrightとtint(t *testing.T) {
	t.Parallel()

	s := lightSample{brightness: 0.7, color: [3]float64{1, 1, 1}}.overbright(1.3)
	assert.InDelta(t, 0.91, s.brightness, 1e-9, "brightness *= boost")
	tint := s.tint()
	assert.InDelta(t, 0.91, tint[0], 1e-9, "tint = 色 × 明るさ")
	assert.InDelta(t, 0.91, tint[2], 1e-9)
}

// TestNormalizeLight は光源色の正規化と、無効時に白へフォールバックすることを固定する。
func TestNormalizeLight(t *testing.T) {
	t.Parallel()
	t.Run("最大成分で正規化し色味だけ残す", func(t *testing.T) {
		t.Parallel()
		got := normalizeLight(color.RGBA{R: 255, G: 128, B: 0, A: 255})
		assert.InDelta(t, 1.0, got[0], 1e-9)
		assert.InDelta(t, 128.0/255, got[1], 1e-9)
		assert.InDelta(t, 0.0, got[2], 1e-9)
	})
	t.Run("アルファ0は白を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, [3]float64{1, 1, 1}, normalizeLight(color.RGBA{R: 255, A: 0}))
	})
	t.Run("黒は白を返す", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, [3]float64{1, 1, 1}, normalizeLight(color.RGBA{A: 255}))
	})
}

// TestRender3DSystem_String は w.Renderer の識別名を固定する。
func TestRender3DSystem_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Render3DSystem", (&Render3DSystem{}).String())
}

// TestVisTintFunc_FOV無効は全タイルを白で描く は視界無効時の分岐を固定する。
func TestVisTintFunc_FOV無効は全タイルを白で描く(t *testing.T) {
	t.Parallel()
	sys := &Render3DSystem{UseFOV: false}
	tint, drawable, visible := sys.visTintFunc(w.World{})(&gc.GridElement{})
	assert.Equal(t, [3]float64{1, 1, 1}, tint)
	assert.True(t, drawable)
	assert.True(t, visible)
}

func TestVisTintFunc_受ける環境光が強いタイルだけを持ち上げる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	outdoor := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 0, Y: 0}}
	indoor := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 0}}
	unlit := gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 2, Y: 0}}
	vs := query.GetVisionState(world)
	vs.VisibleTiles = map[gc.GridElement]bool{outdoor: true, indoor: true, unlit: true}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	vs.LightSourceCache = map[gc.GridElement]gc.LightInfo{
		outdoor: {Color: white, Ambient: 0.95},
		indoor:  {Color: white, Ambient: shelteredAmbient(0.95, 25)},
	}

	tintOf := (&Render3DSystem{UseFOV: true}).visTintFunc(world)
	got, _, _ := tintOf(&outdoor)
	assert.InDeltaSlice(t, []float64{1 + dayOverbrightMax, 1 + dayOverbrightMax, 1 + dayOverbrightMax}, got[:], 1e-9)
	got, _, _ = tintOf(&indoor)
	assert.InDeltaSlice(t, []float64{1, 1, 1}, got[:], 1e-9)
	got, _, _ = tintOf(&unlit)
	brightness := 1 - float64(DarknessVisible)
	assert.InDeltaSlice(t, []float64{brightness, brightness, brightness}, got[:], 1e-9, "光の情報が無いタイルは底上げを受けない")
}

// TestSortQuadsByDepth_奥から手前へ並べる は画家アルゴリズムの前段ソートを固定する。
func TestSortQuadsByDepth_奥から手前へ並べる(t *testing.T) {
	t.Parallel()
	mk := func(z float64) r3quad {
		return r3quad{p: [4]render3d.Vec{render3d.At(0, 0, z), render3d.At(0, 0, z), render3d.At(0, 0, z), render3d.At(0, 0, z)}}
	}
	quads := []r3quad{mk(5), mk(-3), mk(1)}
	sortQuadsByDepth(quads, func(v render3d.Vec) float64 { return v.Z }) // 奥行きは重心zそのもの
	assert.Equal(t, -3.0, quads[0].p[0].Z)
	assert.Equal(t, 1.0, quads[1].p[0].Z)
	assert.Equal(t, 5.0, quads[2].p[0].Z)
}

// 同一タイルの立て板は4隅が一致して奥行きが同値になる。走査順ではなく depth で前後を確定し、
// 大きい depth を後に、つまり手前に描くことを固定する。プレイヤーが足元のアイテムより上に出る根拠
func TestSortQuadsByDepth_奥行き同値はdepthの大きい方を手前にする(t *testing.T) {
	t.Parallel()
	mk := func(depth int) r3quad {
		// 全 quad を同一座標に置き key をビット同値にする。差は depth だけ
		return r3quad{p: [4]render3d.Vec{render3d.At(1, 0, 1), render3d.At(1, 0, 1), render3d.At(1, 0, 1), render3d.At(1, 0, 1)}, depth: depth}
	}
	// アイテム(1)を先、プレイヤー(3)を後に積んでも、逆に積んでも同じ順に並ぶことを確かめる
	quads := []r3quad{mk(1), mk(3)}
	sortQuadsByDepth(quads, func(v render3d.Vec) float64 { return v.Z })
	assert.Equal(t, 1, quads[0].depth, "小さい depth が先で奥")
	assert.Equal(t, 3, quads[1].depth, "大きい depth が後で手前")

	reversed := []r3quad{mk(3), mk(1)}
	sortQuadsByDepth(reversed, func(v render3d.Vec) float64 { return v.Z })
	assert.Equal(t, 1, reversed[0].depth, "積む順に依らず depth 昇順で並ぶ")
	assert.Equal(t, 3, reversed[1].depth)
}

// TestDeployFieldArea は展開野営のタイル集合を投影・描画から切り離して固定する。
// 壁は展開時に凍結した Deployed.Range で描くので、半径 2x2 なら中心の周り 5x5=25 タイルになる。
func TestDeployFieldArea(t *testing.T) {
	t.Parallel()

	t.Run("展開中キューブは凍結範囲の矩形を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube := world.ECS.NewEntity()
		world.Components.GridElement.Add(cube, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
		world.Components.Deployed.Add(cube, &gc.Deployed{Range: consts.Coord[consts.Tile]{X: 2, Y: 2}})

		area := deployFieldArea(world)

		assert.Len(t, area, 25, "半径 2x2 なので 5x5 タイル")
		assert.True(t, area[consts.Coord[consts.Tile]{X: 10, Y: 10}], "中心を含む")
		assert.True(t, area[consts.Coord[consts.Tile]{X: 12, Y: 12}], "角(+2,+2)を含む")
		assert.True(t, area[consts.Coord[consts.Tile]{X: 8, Y: 8}], "角(-2,-2)を含む")
		assert.False(t, area[consts.Coord[consts.Tile]{X: 13, Y: 10}], "範囲外(+3,0)は含まない")
	})

	t.Run("展開していないキューブは空を返す", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube := world.ECS.NewEntity()
		world.Components.GridElement.Add(cube, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})

		assert.Empty(t, deployFieldArea(world), "Deployed が無ければ野営は無い")
	})
}

// TestCollectDeployField は野営エリアの外周だけにレーザー壁クアッドを積むことを固定する。
// deployFieldArea 自体は TestDeployFieldArea が固定済みなので、ここでは projector の画面内判定を
// 通した上でクアッドが実際に積まれること、積まれた色と透明度が固定値であることを見る。
func TestCollectDeployField(t *testing.T) {
	t.Parallel()

	center := consts.Coord[consts.Tile]{X: 25, Y: 25}
	// 注視点を野営の中心に合わせ、範囲内のタイルが必ず画面に掛かるようにする
	projector := render3d.NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, center, 960, 720)

	t.Run("展開中キューブが無ければquadsは変化しない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		sys := &Render3DSystem{}

		quads := sys.collectDeployField(world, nil, projector)

		assert.Empty(t, quads, "野営エリアが無いので何も積まない")
	})

	t.Run("展開中キューブの周囲にレーザー壁を積む", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		cube, err := lifecycle.SpawnCube(world, center)
		require.NoError(t, err)
		require.True(t, lifecycle.DeployCube(world, cube), "野営の展開に成功する")
		sys := &Render3DSystem{}

		quads := sys.collectDeployField(world, nil, projector)

		require.NotEmpty(t, quads, "外周タイルにレーザー壁が積まれる")
		for _, q := range quads {
			assert.Equal(t, deployFieldColor, q.col, "レーザー壁は固定の青で光る")
			assert.InDelta(t, deployFieldAlpha, q.alpha, 1e-9, "エネルギー幕なので半透明")
		}
	})

	t.Run("画面外の野営エリアは積まない", func(t *testing.T) {
		t.Parallel()
		world := testutil.InitTestWorld(t)
		// 500タイルはprojectorの可視距離far=200を大きく超えるので、画面内判定で確実に落ちる
		far := consts.Coord[consts.Tile]{X: center.X + 500, Y: center.Y + 500}
		cube, err := lifecycle.SpawnCube(world, far)
		require.NoError(t, err)
		require.True(t, lifecycle.DeployCube(world, cube), "野営の展開に成功する")
		sys := &Render3DSystem{}

		quads := sys.collectDeployField(world, nil, projector)

		assert.Empty(t, quads, "画面に掛からないタイルはTileOnScreenで落ちる")
	})
}

func TestCollectBillboards_寝かせる形のpropは地面に寝かせる(t *testing.T) {
	t.Parallel()
	center := consts.Coord[consts.Tile]{X: 25, Y: 25}
	projector := render3d.NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, center, 960, 720)
	visible := func(*gc.GridElement) ([3]float64, bool, bool) { return [3]float64{1, 1, 1}, true, true }

	world := testutil.InitTestWorld(t)
	world.Resources.SetSpriteSheets(map[string]gc.SpriteSheet{
		"field": {
			Texture: gc.Texture{Image: ebiten.NewImage(32, 32)},
			Sprites: map[string]gc.Sprite{"soil_crack": {Width: 32, Height: 32}},
		},
	})
	_, err := lifecycle.SpawnProp(world, "soil_crack", center.X, center.Y)
	require.NoError(t, err)
	sys := &Render3DSystem{}

	quads := sys.collectBillboards(world, nil, projector, visible, nil, nil, nil)

	require.Len(t, quads, 1)
	assert.Equal(t, [4]render3d.Vec{
		render3d.At(25, 0, 25), render3d.At(26, 0, 25), render3d.At(26, 0, 26), render3d.At(25, 0, 26),
	}, quads[0].p)
	assert.Equal(t, rawschema.SpriteDepthDecal.Rank(), quads[0].depth)
}

func TestCollectBillboards_視界の外では跡を描かず床の模様は描く(t *testing.T) {
	t.Parallel()
	center := consts.Coord[consts.Tile]{X: 25, Y: 25}
	projector := render3d.NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, center, 960, 720)
	remembered := func(*gc.GridElement) ([3]float64, bool, bool) { return [3]float64{1, 1, 1}, true, false }

	world := testutil.InitTestWorld(t)
	world.Resources.SetSpriteSheets(map[string]gc.SpriteSheet{
		"field": {
			Texture: gc.Texture{Image: ebiten.NewImage(32, 32)},
			Sprites: map[string]gc.Sprite{"soil_crack": {Width: 32, Height: 32}, "oil_stain": {Width: 32, Height: 32}},
		},
	})
	_, err := lifecycle.SpawnProp(world, "soil_crack", center.X, center.Y)
	require.NoError(t, err)
	_, err = lifecycle.SpawnRemains(world, "oil_stain", consts.Coord[consts.Tile]{X: center.X + 1, Y: center.Y})
	require.NoError(t, err)
	sys := &Render3DSystem{}

	quads := sys.collectBillboards(world, nil, projector, remembered, nil, nil, nil)

	require.Len(t, quads, 1, "記憶しているマスには床の模様だけを描く")
	assert.Equal(t, rawschema.SpriteDepthDecal.Rank(), quads[0].depth)
}

func TestSnowLayer_道は行の解けやすさの分だけ薄く積もる(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	query.GetGameTime(world).TotalTurns = 24*1500 + 500
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	query.EnsureSeamlessBand(world)
	row, err := raw.GetTile(world.Resources.RawMaster, consts.TileNameRoad)
	require.NoError(t, err)
	require.NotNil(t, row.SnowMeltPercent, "前提: 道の行が解けやすさを持つ")
	tile, err := lifecycle.SpawnTile(world, consts.TileNameRoad, 3, 3, nil)
	require.NoError(t, err)

	layer := (&Render3DSystem{}).newSnowLayer(world)
	assert.InDelta(t, float64(100-*row.SnowMeltPercent)/100, layer.amount(world, tile, consts.Coord[consts.Tile]{X: 3, Y: 3}), 1e-9)
}

func TestSnowLayer_屋根の無い通れる地面にだけ量を重ねる(t *testing.T) {
	t.Parallel()

	const winter = 24*1500 + 500
	tests := []struct {
		name    string
		turns   consts.Turn
		tile    string
		shelter *rawschema.Shelter
		melt    int
		want    float64
	}{
		{"冬の屋外の土は全面に積もる", winter, consts.TileNameDirt, nil, 0, 1},
		{"冬の屋外でも解けやすさの分だけ薄く積もる", winter, consts.TileNameDirt, nil, 75, 0.25},
		{"冬でも屋内の床には積もらない", winter, "floor_tile", nil, 0, 0},
		{"冬でも通れない壁には積もらない", winter, "dwall", nil, 0, 0},
		{"冬でも屋根のある半屋外には積もらない", winter, consts.TileNameDirt, new(rawschema.Partial), 0, 0},
		{"春は積もらない", 0, consts.TileNameDirt, nil, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			query.GetGameTime(world).TotalTurns = tt.turns
			query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
			query.EnsureSeamlessBand(world)
			tile, err := lifecycle.SpawnTile(world, tt.tile, 3, 3, nil)
			require.NoError(t, err)
			if tt.shelter != nil {
				world.Components.TileEnvironment.Get(tile).Shelter = *tt.shelter
			}
			world.Components.TileEnvironment.Get(tile).SnowMeltPercent = tt.melt

			layer := (&Render3DSystem{}).newSnowLayer(world)
			assert.InDelta(t, tt.want, layer.amount(world, tile, consts.Coord[consts.Tile]{X: 3, Y: 3}), 1e-9)
		})
	}
}

// assertFacesInDelta は面の並びを、頂点を近似で、塗り方を一致で比べる
func assertFacesInDelta(t *testing.T, want, got []insetFace, msg string) {
	t.Helper()
	require.Len(t, got, len(want), msg)
	for i := range want {
		assert.Equal(t, want[i].flat, got[i].flat, msg)
		for j := range want[i].p {
			assert.InDelta(t, want[i].p[j].X, got[i].p[j].X, 1e-9, msg)
			assert.InDelta(t, want[i].p[j].Y, got[i].p[j].Y, 1e-9, msg)
			assert.InDelta(t, want[i].p[j].Z, got[i].p[j].Z, 1e-9, msg)
		}
	}
}

func TestInsetConnections_隣の壁と閉じた壁にはめる置物につながる(t *testing.T) {
	t.Parallel()

	walls := map[consts.Coord[consts.Tile]]bool{{X: 24, Y: 25}: true}
	insets := map[consts.Coord[consts.Tile]]bool{{X: 26, Y: 25}: true}
	assert.Equal(t, connections{east: true, west: true}, insetConnections(walls, insets, consts.Coord[consts.Tile]{X: 25, Y: 25}))
}

func TestInsetFaces_つながる向きに並ぶ薄い板になる(t *testing.T) {
	t.Parallel()

	c := consts.Coord[consts.Tile]{X: 25, Y: 25}
	top := insetFace{p: [4]render3d.Vec{render3d.At(25, 1, 25.4), render3d.At(26, 1, 25.4), render3d.At(26, 1, 25.6), render3d.At(25, 1, 25.6)}, flat: true}
	north := insetFace{p: [4]render3d.Vec{render3d.At(26, 1, 25.4), render3d.At(25, 1, 25.4), render3d.At(25, 0, 25.4), render3d.At(26, 0, 25.4)}}
	south := insetFace{p: [4]render3d.Vec{render3d.At(25, 1, 25.6), render3d.At(26, 1, 25.6), render3d.At(26, 0, 25.6), render3d.At(25, 0, 25.6)}}
	nsTop := insetFace{p: [4]render3d.Vec{render3d.At(25.6, 1, 25), render3d.At(25.6, 1, 26), render3d.At(25.4, 1, 26), render3d.At(25.4, 1, 25)}, flat: true}
	east := insetFace{p: [4]render3d.Vec{render3d.At(25.6, 1, 26), render3d.At(25.6, 1, 25), render3d.At(25.6, 0, 25), render3d.At(25.6, 0, 26)}}
	west := insetFace{p: [4]render3d.Vec{render3d.At(25.4, 1, 25), render3d.At(25.4, 1, 26), render3d.At(25.4, 0, 26), render3d.At(25.4, 0, 25)}}
	westEnd := insetFace{p: [4]render3d.Vec{render3d.At(25, 1, 25.4), render3d.At(25, 1, 25.6), render3d.At(25, 0, 25.6), render3d.At(25, 0, 25.4)}, flat: true}
	cases := []struct {
		name string
		conn connections
		want []insetFace
	}{
		{"東西につながれば東西に張り、両端を省く", connections{east: true, west: true}, []insetFace{top, north, south}},
		{"南北につながれば南北に張る", connections{north: true, south: true}, []insetFace{nsTop, east, west}},
		{"北だけにつながれば南の端を描く", connections{north: true}, []insetFace{nsTop, east, west,
			{p: [4]render3d.Vec{render3d.At(25.4, 1, 26), render3d.At(25.6, 1, 26), render3d.At(25.6, 0, 26), render3d.At(25.4, 0, 26)}, flat: true}}},
		{"東西に片側でもつながれば南北より優先して東西に張る", connections{north: true, east: true, south: true}, []insetFace{top, north, south, westEnd}},
		{"つながりが無ければ東西に張って両端を描く", connections{}, []insetFace{top, north, south, westEnd,
			{p: [4]render3d.Vec{render3d.At(26, 1, 25.6), render3d.At(26, 1, 25.4), render3d.At(26, 0, 25.4), render3d.At(26, 0, 25.6)}, flat: true}}},
	}
	for _, tc := range cases {
		assertFacesInDelta(t, tc.want, insetFaces(c, tc.conn), tc.name)
	}
}

func TestFaceShade_天面と東西と南北で陰影を分ける(t *testing.T) {
	t.Parallel()

	assert.InDelta(t, 0.95, faceShade([4]render3d.Vec{render3d.At(0, 1, 0), render3d.At(1, 1, 0), render3d.At(1, 1, 1), render3d.At(0, 1, 1)}), 1e-9)
	assert.InDelta(t, 0.78, faceShade([4]render3d.Vec{render3d.At(0, 1, 0), render3d.At(0, 1, 1), render3d.At(0, 0, 1), render3d.At(0, 0, 0)}), 1e-9)
	assert.InDelta(t, 0.6, faceShade([4]render3d.Vec{render3d.At(1, 1, 0), render3d.At(0, 1, 0), render3d.At(0, 0, 0), render3d.At(1, 0, 0)}), 1e-9)
}

func TestClosedInsets_閉じた壁にはめる置物だけを集める(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 3, Y: 3}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	opened, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 4, Y: 3}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	require.NoError(t, lifecycle.OpenDoor(world, opened))
	_, err = lifecycle.SpawnProp(world, "window", 5, 3)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "barrel", 6, 3)
	require.NoError(t, err)

	assert.Equal(t, map[consts.Coord[consts.Tile]]bool{{X: 3, Y: 3}: true, {X: 5, Y: 3}: true}, closedInsets(world))
}

func TestCollectBillboards_開いた扉は描かない(t *testing.T) {
	t.Parallel()
	center := consts.Coord[consts.Tile]{X: 25, Y: 25}
	projector := render3d.NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, center, 960, 720)
	visible := func(*gc.GridElement) ([3]float64, bool, bool) { return [3]float64{1, 1, 1}, true, true }

	world := testutil.InitTestWorld(t)
	world.Resources.SetSpriteSheets(map[string]gc.SpriteSheet{
		"field": {
			Texture: gc.Texture{Image: ebiten.NewImage(32, 32)},
			Sprites: map[string]gc.Sprite{"door_vertical_closed": {Width: 32, Height: 32}},
		},
	})
	door, err := lifecycle.SpawnDoor(world, center, gc.DoorOrientationHorizontal)
	require.NoError(t, err)
	require.NoError(t, lifecycle.OpenDoor(world, door))
	sys := &Render3DSystem{}

	assert.Empty(t, sys.collectBillboards(world, nil, projector, visible, nil, closedInsets(world), nil))
}

func TestFacingFaces_閉じた板の正面と背面に絵を貼る(t *testing.T) {
	t.Parallel()

	c := consts.Coord[consts.Tile]{X: 5, Y: 5}
	westBacked := gc.Pose{Facing: consts.West, Anchor: gc.AnchorBack}
	// 北の隣と接するので北の端を省く
	all := func(consts.Coord[consts.Tile]) bool { return true }
	faces := facingFaces(c, westBacked, map[consts.Coord[consts.Tile]]gc.Pose{c: westBacked, {X: 5, Y: 4}: westBacked}, all)
	require.Len(t, faces, 7)
	for _, f := range faces[:5] {
		assert.True(t, f.flat)
		for _, v := range f.p {
			assert.True(t, v.X >= 5.78-1e-9 && v.X <= 5.98+1e-9 && v.Z >= 5-1e-9 && v.Z <= 6+1e-9, "板は東の壁へ寄せた範囲に収まる %v", v)
		}
	}
	assertFacesInDelta(t, []insetFace{
		{p: [4]render3d.Vec{render3d.At(5.77, 1, 5), render3d.At(5.77, 1, 6), render3d.At(5.77, 0, 6), render3d.At(5.77, 0, 5)}},
		{p: [4]render3d.Vec{render3d.At(5.99, 1, 6), render3d.At(5.99, 1, 5), render3d.At(5.99, 0, 5), render3d.At(5.99, 0, 6)}},
	}, faces[5:], "絵は板の西の正面と東の背面から、外向きに少し浮かせて貼る")

	alone := gc.Pose{Facing: consts.South}
	assert.Len(t, facingFaces(c, alone, map[consts.Coord[consts.Tile]]gc.Pose{c: alone}, all), 8)

	north := gc.Pose{Facing: consts.North}
	assertFacesInDelta(t, []insetFace{
		{p: [4]render3d.Vec{render3d.At(6, 1, 5.39), render3d.At(5, 1, 5.39), render3d.At(5, 0, 5.39), render3d.At(6, 0, 5.39)}},
	}, facingFaces(c, north, map[consts.Coord[consts.Tile]]gc.Pose{c: north}, all)[6:7], "北向きの正面")
	east := gc.Pose{Facing: consts.East}
	assertFacesInDelta(t, []insetFace{
		{p: [4]render3d.Vec{render3d.At(5.61, 1, 6), render3d.At(5.61, 1, 5), render3d.At(5.61, 0, 5), render3d.At(5.61, 0, 6)}},
	}, facingFaces(c, east, map[consts.Coord[consts.Tile]]gc.Pose{c: east}, all)[6:7], "東向きの正面")

	// 北の隣が視界の外で描かれなければ、北の端を省かず板を閉じる
	hidden := func(n consts.Coord[consts.Tile]) bool { return n != consts.Coord[consts.Tile]{X: 5, Y: 4} }
	assert.Len(t, facingFaces(c, westBacked, map[consts.Coord[consts.Tile]]gc.Pose{c: westBacked, {X: 5, Y: 4}: westBacked}, hidden), 8)

	// 区画の境などで隣が同じ向きでも寄せる位置が違えば、継ぎ目に穴が開かないよう端を閉じる
	assert.Len(t, facingFaces(c, westBacked, map[consts.Coord[consts.Tile]]gc.Pose{c: westBacked, {X: 5, Y: 4}: {Facing: consts.West}}, all), 8)
}

func TestPosesAt_地図を作るときに決めた向きを座標ごとに集める(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	shelf, err := lifecycle.SpawnProp(world, "book_shelf_2", 3, 3)
	require.NoError(t, err)
	world.Components.Pose.Add(shelf, &gc.Pose{Facing: consts.West, Anchor: gc.AnchorBack})
	_, err = lifecycle.SpawnProp(world, "book_shelf_2", 4, 3)
	require.NoError(t, err)

	assert.Equal(t, map[consts.Coord[consts.Tile]]gc.Pose{{X: 3, Y: 3}: {Facing: consts.West, Anchor: gc.AnchorBack}}, posesAt(world))
}

func TestCollectBillboards_同じマスのテーブルとアイテムはアイテムを手前に描く(t *testing.T) {
	t.Parallel()
	center := consts.Coord[consts.Tile]{X: 25, Y: 25}
	projector := render3d.NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, center, 960, 720)
	visible := func(*gc.GridElement) ([3]float64, bool, bool) { return [3]float64{1, 1, 1}, true, true }

	world := testutil.InitTestWorld(t)
	world.Resources.SetSpriteSheets(map[string]gc.SpriteSheet{
		"field": {
			Texture: gc.Texture{Image: ebiten.NewImage(32, 32)},
			Sprites: map[string]gc.Sprite{"dining_table": {Width: 32, Height: 32}, "donut": {Width: 32, Height: 32}},
		},
	})
	_, err := lifecycle.SpawnFieldItem(world, "donut", center.X, center.Y, 1)
	require.NoError(t, err)
	_, err = lifecycle.SpawnProp(world, "table", center.X, center.Y)
	require.NoError(t, err)
	sys := &Render3DSystem{}

	quads := sys.collectBillboards(world, nil, projector, visible, nil, nil, nil)
	sortQuadsByDepth(quads, projector.Depth)

	require.Len(t, quads, 2)
	assert.Equal(t, []int{rawschema.SpriteDepthTaller.Rank(), rawschema.SpriteDepthItem.Rank()}, []int{quads[0].depth, quads[1].depth}, "後に描くアイテムが手前に出る")
}

func TestCollectBillboards_同じマスのアイテムとプレイヤーはプレイヤーを手前に描く(t *testing.T) {
	t.Parallel()
	center := consts.Coord[consts.Tile]{X: 25, Y: 25}
	projector := render3d.NewProjector(gc.Camera{Pitch: gc.CameraDefaultPitch, Dist: gc.CameraDefaultDist}, center, 960, 720)
	visible := func(*gc.GridElement) ([3]float64, bool, bool) { return [3]float64{1, 1, 1}, true, true }

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, center, "ash")
	require.NoError(t, err)
	playerKey := world.Components.SpriteRender.Get(player).SpriteKey
	world.Resources.SetSpriteSheets(map[string]gc.SpriteSheet{
		"field": {
			Texture: gc.Texture{Image: ebiten.NewImage(32, 32)},
			Sprites: map[string]gc.Sprite{playerKey: {Width: 32, Height: 32}, "donut": {Width: 32, Height: 32}},
		},
	})
	_, err = lifecycle.SpawnFieldItem(world, "donut", center.X, center.Y, 1)
	require.NoError(t, err)
	sys := &Render3DSystem{}

	quads := sys.collectBillboards(world, nil, projector, visible, nil, nil, nil)
	sortQuadsByDepth(quads, projector.Depth)

	require.Len(t, quads, 2)
	assert.Equal(t, []int{rawschema.SpriteDepthItem.Rank(), rawschema.SpriteDepthPlayer.Rank()}, []int{quads[0].depth, quads[1].depth}, "後に描くプレイヤーが手前に出る")
}
