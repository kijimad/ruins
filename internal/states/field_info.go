package states

import (
	"fmt"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	gc "github.com/kijimaD/ruins/internal/components"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/input"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FieldInfoState は視界情報確認画面のステート
// 視界内の敵やアイテムの情報を表示し、特性や距離を確認できる
type FieldInfoState struct {
	es.BaseState[w.World]
	entries       []fieldEntry
	selectedIndex int
}

func (st FieldInfoState) String() string {
	return "FieldInfo"
}

var _ es.State[w.World] = &FieldInfoState{}

// OnPause はステートが一時停止される際に呼ばれる
func (st *FieldInfoState) OnPause(_ w.World) error { return nil }

// OnResume はステートが再開される際に呼ばれる
func (st *FieldInfoState) OnResume(_ w.World) error { return nil }

// OnStart はステートが開始される際に呼ばれる
func (st *FieldInfoState) OnStart(world w.World) error {
	playerEntity, err := worldhelper.GetPlayerEntity(world)
	if err != nil {
		return err
	}

	if err := st.collectEnemyEntries(world, playerEntity); err != nil {
		return err
	}

	if err := st.collectItemEntries(world, playerEntity); err != nil {
		return err
	}

	// 距離順にソート
	sort.Slice(st.entries, func(i, j int) bool {
		return st.entries[i].Distance < st.entries[j].Distance
	})

	return nil
}

// OnStop はステートが終了する際に呼ばれる
func (st *FieldInfoState) OnStop(_ w.World) error { return nil }

// Update はステートの更新処理
func (st *FieldInfoState) Update(_ w.World) (es.Transition[w.World], error) {
	keyboardInput := input.GetSharedKeyboardInput()

	// Escapeキーで終了
	if keyboardInput.IsKeyJustPressed(ebiten.KeyEscape) {
		return es.Transition[w.World]{Type: es.TransPop}, nil
	}

	if len(st.entries) == 0 {
		return es.Transition[w.World]{Type: es.TransNone}, nil
	}

	// 上下キーで選択
	if keyboardInput.IsKeyJustPressed(ebiten.KeyUp) || keyboardInput.IsKeyJustPressed(ebiten.KeyW) {
		st.selectedIndex--
		if st.selectedIndex < 0 {
			st.selectedIndex = len(st.entries) - 1
		}
	}
	if keyboardInput.IsKeyJustPressed(ebiten.KeyDown) || keyboardInput.IsKeyJustPressed(ebiten.KeyS) {
		st.selectedIndex++
		if st.selectedIndex >= len(st.entries) {
			st.selectedIndex = 0
		}
	}

	// BaseStateの共通処理を使用
	return st.ConsumeTransition(), nil
}

// Draw はステートの描画処理
func (st *FieldInfoState) Draw(world w.World, screen *ebiten.Image) error {
	face := world.Resources.UIResources.Text.BodyFace

	const (
		marginX       = 40
		marginY       = 60
		lineHeight    = 24
		sectionMargin = 16
	)

	// drawText はテキストを描画するヘルパー関数
	drawText := func(str string, y int) {
		op := &text.DrawOptions{}
		op.GeoM.Translate(float64(marginX), float64(y))
		text.Draw(screen, str, face, op)
	}

	y := marginY

	if len(st.entries) == 0 {
		drawText("視界内に何もありません", y)
		return nil
	}

	// エントリリスト
	drawText("視界内の情報 (距離順):", y)
	y += lineHeight + 8

	for i, entry := range st.entries {
		// 選択中のエントリを強調
		prefix := "  "
		if i == st.selectedIndex {
			prefix = "> "
		}

		// タイプ表示
		typeStr := "[敵]"
		if entry.Type == "item" {
			typeStr = "[物]"
		}

		// エントリ情報を表示
		var line string
		if entry.Type == "enemy" {
			line = fmt.Sprintf("%s%d. %s %-12s  距離:%2d  HP:%d/%d",
				prefix, i+1, typeStr, entry.Name, entry.Distance, entry.HP, entry.MaxHP)
		} else {
			line = fmt.Sprintf("%s%d. %s %-12s  距離:%2d",
				prefix, i+1, typeStr, entry.Name, entry.Distance)
		}
		drawText(line, y)
		y += lineHeight
	}

	// 選択中のエントリの詳細情報
	y += sectionMargin
	drawText("================================", y)
	y += lineHeight

	// インデックスが範囲内かチェック
	if st.selectedIndex < 0 || st.selectedIndex >= len(st.entries) {
		return fmt.Errorf("選択インデックスが範囲外です: %d (範囲: 0-%d)", st.selectedIndex, len(st.entries)-1)
	}

	selected := st.entries[st.selectedIndex]
	drawText(fmt.Sprintf("名前: %s", selected.Name), y)
	y += lineHeight
	if selected.Type == "enemy" {
		drawText(fmt.Sprintf("HP: %d/%d", selected.HP, selected.MaxHP), y)
		y += lineHeight
	} else if selected.Type == "item" && selected.Description != "" {
		drawText(fmt.Sprintf("説明: %s", selected.Description), y)
		y += lineHeight
	}
	drawText(fmt.Sprintf("距離: %d タイル", selected.Distance), y)
	y += lineHeight
	drawText(fmt.Sprintf("座標: (%d, %d)", selected.GridX, selected.GridY), y)

	// 操作説明
	y = screen.Bounds().Dy() - 80
	drawText("--- 操作 ---", y)
	y += lineHeight
	drawText("↑↓/WS: 移動", y)
	y += lineHeight
	drawText("Esc: 閉じる", y)

	return nil
}

// fieldEntry は視界内のエンティティ情報
type fieldEntry struct {
	Type        string // "enemy" or "item"
	Entity      ecs.Entity
	Name        string
	Description string
	HP          int
	MaxHP       int
	Distance    int
	GridX       int
	GridY       int
}

// collectEnemyEntries は視界内の敵エントリを収集する
func (st *FieldInfoState) collectEnemyEntries(world w.World, playerEntity ecs.Entity) error {
	enemies, err := worldhelper.GetVisibleEnemies(world)
	if err != nil {
		return err
	}

	for _, enemyEntity := range enemies {
		// エンティティから必要な情報を取得
		name := worldhelper.GetEntityName(enemyEntity, world)
		distance := worldhelper.CalculateDistance(world, playerEntity, enemyEntity)

		var hp, maxHP int
		if enemyEntity.HasComponent(world.Components.Pools) {
			pools := world.Components.Pools.Get(enemyEntity).(*gc.Pools)
			hp = pools.HP.Current
			maxHP = pools.HP.Max
		}

		var gridX, gridY int
		if enemyEntity.HasComponent(world.Components.GridElement) {
			grid := world.Components.GridElement.Get(enemyEntity).(*gc.GridElement)
			gridX = int(grid.X)
			gridY = int(grid.Y)
		}

		st.entries = append(st.entries, fieldEntry{
			Type:     "enemy",
			Entity:   enemyEntity,
			Name:     name,
			HP:       hp,
			MaxHP:    maxHP,
			Distance: distance,
			GridX:    gridX,
			GridY:    gridY,
		})
	}

	return nil
}

// collectItemEntries は視界内のアイテムエントリを収集する
func (st *FieldInfoState) collectItemEntries(world w.World, playerEntity ecs.Entity) error {
	items, err := worldhelper.GetVisibleItems(world)
	if err != nil {
		return err
	}

	for _, itemEntity := range items {
		// エンティティから必要な情報を取得
		name := worldhelper.GetEntityName(itemEntity, world)
		distance := worldhelper.CalculateDistance(world, playerEntity, itemEntity)

		var description string
		if itemEntity.HasComponent(world.Components.Description) {
			desc := world.Components.Description.Get(itemEntity).(*gc.Description)
			description = desc.Description
		}

		var gridX, gridY int
		if itemEntity.HasComponent(world.Components.GridElement) {
			grid := world.Components.GridElement.Get(itemEntity).(*gc.GridElement)
			gridX = int(grid.X)
			gridY = int(grid.Y)
		}

		st.entries = append(st.entries, fieldEntry{
			Type:        "item",
			Entity:      itemEntity,
			Name:        name,
			Description: description,
			Distance:    distance,
			GridX:       gridX,
			GridY:       gridY,
		})
	}

	return nil
}
