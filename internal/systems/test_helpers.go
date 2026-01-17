package systems

import (
	"fmt"
	"testing"

	"github.com/kijimaD/ruins/internal/engine/entities"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/stretchr/testify/require"

	gc "github.com/kijimaD/ruins/internal/components"
)

// spriteSize はテスト用のスプライトサイズ構造体
type spriteSize struct {
	width  int
	height int
}

// CreatePlayerEntity は指定された位置にプレイヤーエンティティを作成する
func CreatePlayerEntity(t *testing.T, world w.World, x, y float64) {
	t.Helper()

	// テスト用のスプライトシートを作成してResourcesに追加
	if world.Resources.SpriteSheets == nil {
		sheets := make(map[string]gc.SpriteSheet)
		world.Resources.SpriteSheets = &sheets
	}
	(*world.Resources.SpriteSheets)["test"] = gc.SpriteSheet{
		Name: "test",
		Sprites: map[string]gc.Sprite{
			"test_sprite": {Width: 32, Height: 32}, // 標準サイズ
		},
	}

	cl := entities.ComponentList[gc.EntitySpec]{}
	cl.Entities = append(cl.Entities, gc.EntitySpec{
		Position:    &gc.Position{X: gc.Pixel(x), Y: gc.Pixel(y)},
		FactionType: &gc.FactionAlly,
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: "test",
			SpriteKey:       "test_sprite",
		},
	})
	_, err := entities.AddEntities(world, cl)
	require.NoError(t, err)
}

// CreateEnemyEntity は指定された位置に敵エンティティを作成する
func CreateEnemyEntity(t *testing.T, world w.World, x, y float64) {
	t.Helper()

	// テスト用のスプライトシートを作成してResourcesに追加
	if world.Resources.SpriteSheets == nil {
		sheets := make(map[string]gc.SpriteSheet)
		world.Resources.SpriteSheets = &sheets
	}
	(*world.Resources.SpriteSheets)["test"] = gc.SpriteSheet{
		Name: "test",
		Sprites: map[string]gc.Sprite{
			"test_sprite": {Width: 32, Height: 32}, // 標準サイズ
		},
	}

	cl := entities.ComponentList[gc.EntitySpec]{}
	cl.Entities = append(cl.Entities, gc.EntitySpec{
		Position:  &gc.Position{X: gc.Pixel(x), Y: gc.Pixel(y)},
		AIMoveFSM: &gc.AIMoveFSM{}, // AI制御された敵として識別
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: "test",
			SpriteKey:       "test_sprite",
		},
	})
	_, err := entities.AddEntities(world, cl)
	require.NoError(t, err)
}

// CreateEntityWithSprite は指定されたスプライトサイズでエンティティを作成する
func CreateEntityWithSprite(t *testing.T, world w.World, x, y float64, width, height int, isPlayer bool) {
	t.Helper()

	// 一意なスプライトシート名を生成
	var sheetName string
	if isPlayer {
		sheetName = fmt.Sprintf("player_%dx%d", width, height)
	} else {
		sheetName = fmt.Sprintf("enemy_%dx%d", width, height)
	}

	// テスト用のスプライトシートを作成してResourcesに追加
	if world.Resources.SpriteSheets == nil {
		sheets := make(map[string]gc.SpriteSheet)
		world.Resources.SpriteSheets = &sheets
	}
	(*world.Resources.SpriteSheets)[sheetName] = gc.SpriteSheet{
		Name: sheetName,
		Sprites: map[string]gc.Sprite{
			"test_sprite": {Width: width, Height: height}, // テスト用スプライト
		},
	}

	// テスト用のスプライト情報を作成
	spriteRender := &gc.SpriteRender{
		SpriteSheetName: sheetName,
		SpriteKey:       "test_sprite",
	}

	cl := entities.ComponentList[gc.EntitySpec]{}
	if isPlayer {
		cl.Entities = append(cl.Entities, gc.EntitySpec{
			Position:     &gc.Position{X: gc.Pixel(x), Y: gc.Pixel(y)},
			FactionType:  &gc.FactionAlly,
			SpriteRender: spriteRender,
		})
	} else {
		cl.Entities = append(cl.Entities, gc.EntitySpec{
			Position:     &gc.Position{X: gc.Pixel(x), Y: gc.Pixel(y)},
			FactionType:  &gc.FactionEnemy,
			SpriteRender: spriteRender,
		})
	}

	_, err := entities.AddEntities(world, cl)
	require.NoError(t, err)
}

// CreateEntityWithSpriteSize はspriteSize構造体を使ってエンティティを作成する
func CreateEntityWithSpriteSize(t *testing.T, world w.World, x, y float64, size spriteSize, isPlayer bool) {
	t.Helper()
	CreateEntityWithSprite(t, world, x, y, size.width, size.height, isPlayer)
}

// CreateEntityWithGridPosition はグリッド座標でエンティティを作成する
func CreateEntityWithGridPosition(t *testing.T, world w.World, gridX, gridY int, isPlayer bool) {
	t.Helper()

	var cl entities.ComponentList[gc.EntitySpec]

	if isPlayer {
		cl.Entities = append(cl.Entities, gc.EntitySpec{
			GridElement: &gc.GridElement{X: gc.Tile(gridX), Y: gc.Tile(gridY)},
			FactionType: &gc.FactionAlly,
		})
	} else {
		cl.Entities = append(cl.Entities, gc.EntitySpec{
			GridElement: &gc.GridElement{X: gc.Tile(gridX), Y: gc.Tile(gridY)},
			AIMoveFSM:   &gc.AIMoveFSM{},
			FactionType: &gc.FactionEnemy,
		})
	}

	_, err := entities.AddEntities(world, cl)
	require.NoError(t, err)
}
