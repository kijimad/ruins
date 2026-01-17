package save

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ecs "github.com/x-hgg-x/goecs/v2"
)

func TestSaveLoadSpriteRender(t *testing.T) {
	t.Parallel()
	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "save_test_")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// ワールドを作成
	w := testutil.InitTestWorld(t)

	// テスト用のスプライトシートを作成してリソースに追加
	testImage := ebiten.NewImage(64, 64)
	testSpriteSheet := gc.SpriteSheet{
		Name:    "test_sprite",
		Texture: gc.Texture{Image: testImage},
		Sprites: map[string]gc.Sprite{
			"sprite1": {X: 0, Y: 0, Width: 32, Height: 32},
			"sprite2": {X: 32, Y: 0, Width: 32, Height: 32},
		},
	}
	if w.Resources.SpriteSheets == nil {
		sheets := make(map[string]gc.SpriteSheet)
		w.Resources.SpriteSheets = &sheets
	}
	(*w.Resources.SpriteSheets)["test_sprite"] = testSpriteSheet

	// SpriteRenderコンポーネントを持つエンティティを作成
	entity := w.Manager.NewEntity()
	entity.AddComponent(w.Components.Name, &gc.Name{Name: "テストエンティティ"})
	entity.AddComponent(w.Components.SpriteRender, &gc.SpriteRender{
		SpriteSheetName: "test_sprite",
		SpriteKey:       "sprite2",
		Depth:           gc.DepthNum(10),
	})

	// スプライトシートなしのエンティティも作成
	entity2 := w.Manager.NewEntity()
	entity2.AddComponent(w.Components.Name, &gc.Name{Name: "スプライトなしエンティティ"})
	entity2.AddComponent(w.Components.SpriteRender, &gc.SpriteRender{
		SpriteSheetName: "", // スプライトシートなし
		SpriteKey:       "",
		Depth:           gc.DepthNum(5),
	})

	// セーブマネージャーを作成
	sm := NewSerializationManager(tempDir)

	// ワールドを保存
	err = sm.SaveWorld(w, "test_sprite")
	require.NoError(t, err)

	// セーブファイルが存在することを確認
	saveFile := filepath.Join(tempDir, "test_sprite.json")
	_, err = os.Stat(saveFile)
	require.NoError(t, err)

	// 新しいワールドを作成してロード
	newWorld := testutil.InitTestWorld(t)

	// リソースに同じスプライトシートを追加（通常はリソースは別途ロードされる）
	if newWorld.Resources.SpriteSheets == nil {
		sheets := make(map[string]gc.SpriteSheet)
		newWorld.Resources.SpriteSheets = &sheets
	}
	(*newWorld.Resources.SpriteSheets)["test_sprite"] = testSpriteSheet

	err = sm.LoadWorld(newWorld, "test_sprite")
	require.NoError(t, err)

	// SpriteRenderコンポーネントが正しく復元されることを確認
	spriteCount := 0
	newWorld.Manager.Join(
		newWorld.Components.Name,
		newWorld.Components.SpriteRender,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := newWorld.Components.Name.Get(entity).(*gc.Name)
		sprite := newWorld.Components.SpriteRender.Get(entity).(*gc.SpriteRender)

		switch name.Name {
		case "テストエンティティ":
			assert.Equal(t, "test_sprite", sprite.SpriteSheetName, "SpriteSheetNameが正しくない")
			assert.Equal(t, "sprite2", sprite.SpriteKey, "SpriteKeyが正しくない")
			assert.Equal(t, gc.DepthNum(10), sprite.Depth, "Depthが正しくない")
			spriteCount++
		case "スプライトなしエンティティ":
			assert.Equal(t, "", sprite.SpriteSheetName, "SpriteSheetNameが空でない")
			assert.Equal(t, "", sprite.SpriteKey, "SpriteKeyが正しくない")
			assert.Equal(t, gc.DepthNum(5), sprite.Depth, "Depthが正しくない")
			spriteCount++
		}
	}))
	assert.Equal(t, 0, spriteCount, "SpriteRenderを持つフィールドエンティティは保存されない（プレイヤーとアイテムのみ保存）")
}
