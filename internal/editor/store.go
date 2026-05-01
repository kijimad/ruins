package editor

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/kijimaD/ruins/internal/raw"
)

// Store はraw.tomlの読み書きを管理する
type Store struct {
	mu   sync.RWMutex
	path string
	raws raw.Raws
}

// NewStore は指定パスのraw.tomlを読み込んでStoreを作成する
func NewStore(path string) (*Store, error) {
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	bs, err := os.ReadFile(s.path)
	if err != nil {
		return fmt.Errorf("raw.tomlの読み込みに失敗: %w", err)
	}
	var raws raw.Raws
	if _, err := toml.Decode(string(bs), &raws); err != nil {
		return fmt.Errorf("raw.tomlのパースに失敗: %w", err)
	}
	s.raws = raws
	s.sortAll()
	return nil
}

// sortItems はアイテムを種別の組み合わせ順、同種別内では名前順にソートする
func sortItems(items []raw.Item) {
	sort.SliceStable(items, func(i, j int) bool {
		ki, kj := itemSortKey(items[i]), itemSortKey(items[j])
		if ki != kj {
			return ki < kj
		}
		return items[i].Name < items[j].Name
	})
}

// itemSortKey はアイテムの種別フラグの組み合わせからソートキーを生成する。
// 同じ種別の組み合わせを持つアイテムが隣接して並ぶ
func itemSortKey(item raw.Item) string {
	var key []byte
	flags := []struct {
		present bool
		code    byte
	}{
		{item.Weapon != nil, 'A'},
		{item.Melee != nil, 'B'},
		{item.Fire != nil, 'C'},
		{item.Wearable != nil, 'D'},
		{item.Consumable != nil, 'E'},
		{item.Ammo != nil, 'F'},
		{item.Book != nil, 'G'},
	}
	for _, f := range flags {
		if f.present {
			key = append(key, f.code)
		}
	}
	if len(key) == 0 {
		key = append(key, 'Z')
	}
	return string(key)
}

// sortMembers はメンバーを名前順にソートする
func sortMembers(members []raw.Member) {
	sort.SliceStable(members, func(i, j int) bool {
		return members[i].Name < members[j].Name
	})
}

// sortRecipes はレシピを名前順にソートする
func sortRecipes(recipes []raw.Recipe) {
	sort.SliceStable(recipes, func(i, j int) bool {
		return recipes[i].Name < recipes[j].Name
	})
}

func (s *Store) sortAll() {
	sortItems(s.raws.Items)
	sortMembers(s.raws.Members)
	sortRecipes(s.raws.Recipes)
	sortByName(s.raws.CommandTables, func(i int) string { return s.raws.CommandTables[i].Name })
	sortByName(s.raws.DropTables, func(i int) string { return s.raws.DropTables[i].Name })
	sortByName(s.raws.ItemTables, func(i int) string { return s.raws.ItemTables[i].Name })
	sortByName(s.raws.EnemyTables, func(i int) string { return s.raws.EnemyTables[i].Name })
	sortByName(s.raws.SpriteSheets, func(i int) string { return s.raws.SpriteSheets[i].Name })
	sortByName(s.raws.Tiles, func(i int) string { return s.raws.Tiles[i].Name })
	sortByName(s.raws.Props, func(i int) string { return s.raws.Props[i].Name })
	sortByName(s.raws.Professions, func(i int) string { return s.raws.Professions[i].ID })
}

// sortByName は任意のスライスを名前関数で取得した値でソートする
func sortByName[T any](slice []T, name func(int) string) {
	sort.SliceStable(slice, func(i, j int) bool {
		return name(i) < name(j)
	})
}

func (s *Store) save() (retErr error) {
	s.sortAll()
	f, err := os.Create(s.path)
	if err != nil {
		return fmt.Errorf("raw.tomlの書き込みに失敗: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("raw.tomlのクローズに失敗: %w", cerr)
		}
	}()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(s.raws); err != nil {
		return fmt.Errorf("raw.tomlのエンコードに失敗: %w", err)
	}
	return nil
}

// SpriteSheets はスプライトシート一覧を返す
func (s *Store) SpriteSheets() []raw.SpriteSheet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.SpriteSheets
}

// Items はアイテム一覧を返す
func (s *Store) Items() []raw.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Items
}

// Item は指定インデックスのアイテムを返す
func (s *Store) Item(index int) (raw.Item, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Items) {
		return raw.Item{}, fmt.Errorf("アイテムインデックスが範囲外: %d", index)
	}
	return s.raws.Items[index], nil
}

// UpdateItem は指定インデックスのアイテムを更新してファイルに保存する
func (s *Store) UpdateItem(index int, item raw.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Items) {
		return fmt.Errorf("アイテムインデックスが範囲外: %d", index)
	}
	s.raws.Items[index] = item
	return s.save()
}

// AddItem は新しいアイテムを追加してファイルに保存する
func (s *Store) AddItem(item raw.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Items = append(s.raws.Items, item)
	return s.save()
}

// DeleteItem は指定インデックスのアイテムを削除してファイルに保存する
func (s *Store) DeleteItem(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Items) {
		return fmt.Errorf("アイテムインデックスが範囲外: %d", index)
	}
	s.raws.Items = append(s.raws.Items[:index], s.raws.Items[index+1:]...)
	return s.save()
}

// Members はメンバー一覧を返す
func (s *Store) Members() []raw.Member {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Members
}

// Member は指定インデックスのメンバーを返す
func (s *Store) Member(index int) (raw.Member, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Members) {
		return raw.Member{}, fmt.Errorf("メンバーインデックスが範囲外: %d", index)
	}
	return s.raws.Members[index], nil
}

// UpdateMember は指定インデックスのメンバーを更新してファイルに保存する
func (s *Store) UpdateMember(index int, member raw.Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Members) {
		return fmt.Errorf("メンバーインデックスが範囲外: %d", index)
	}
	s.raws.Members[index] = member
	return s.save()
}

// AddMember は新しいメンバーを追加してファイルに保存する
func (s *Store) AddMember(member raw.Member) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Members = append(s.raws.Members, member)
	return s.save()
}

// DeleteMember は指定インデックスのメンバーを削除してファイルに保存する
func (s *Store) DeleteMember(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Members) {
		return fmt.Errorf("メンバーインデックスが範囲外: %d", index)
	}
	s.raws.Members = append(s.raws.Members[:index], s.raws.Members[index+1:]...)
	return s.save()
}

// Recipes はレシピ一覧を返す
func (s *Store) Recipes() []raw.Recipe {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Recipes
}

// Recipe は指定インデックスのレシピを返す
func (s *Store) Recipe(index int) (raw.Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Recipes) {
		return raw.Recipe{}, fmt.Errorf("レシピインデックスが範囲外: %d", index)
	}
	return s.raws.Recipes[index], nil
}

// UpdateRecipe は指定インデックスのレシピを更新してファイルに保存する
func (s *Store) UpdateRecipe(index int, recipe raw.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Recipes) {
		return fmt.Errorf("レシピインデックスが範囲外: %d", index)
	}
	s.raws.Recipes[index] = recipe
	return s.save()
}

// AddRecipe は新しいレシピを追加してファイルに保存する
func (s *Store) AddRecipe(recipe raw.Recipe) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Recipes = append(s.raws.Recipes, recipe)
	return s.save()
}

// DeleteRecipe は指定インデックスのレシピを削除してファイルに保存する
func (s *Store) DeleteRecipe(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Recipes) {
		return fmt.Errorf("レシピインデックスが範囲外: %d", index)
	}
	s.raws.Recipes = append(s.raws.Recipes[:index], s.raws.Recipes[index+1:]...)
	return s.save()
}

// CommandTables はコマンドテーブル一覧を返す
func (s *Store) CommandTables() []raw.CommandTable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.CommandTables
}

// CommandTable は指定インデックスのコマンドテーブルを返す
func (s *Store) CommandTable(index int) (raw.CommandTable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.CommandTables) {
		return raw.CommandTable{}, fmt.Errorf("コマンドテーブルインデックスが範囲外: %d", index)
	}
	return s.raws.CommandTables[index], nil
}

// UpdateCommandTable は指定インデックスのコマンドテーブルを更新してファイルに保存する
func (s *Store) UpdateCommandTable(index int, ct raw.CommandTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.CommandTables) {
		return fmt.Errorf("コマンドテーブルインデックスが範囲外: %d", index)
	}
	s.raws.CommandTables[index] = ct
	return s.save()
}

// AddCommandTable は新しいコマンドテーブルを追加してファイルに保存する
func (s *Store) AddCommandTable(ct raw.CommandTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.CommandTables = append(s.raws.CommandTables, ct)
	return s.save()
}

// DeleteCommandTable は指定インデックスのコマンドテーブルを削除してファイルに保存する
func (s *Store) DeleteCommandTable(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.CommandTables) {
		return fmt.Errorf("コマンドテーブルインデックスが範囲外: %d", index)
	}
	s.raws.CommandTables = append(s.raws.CommandTables[:index], s.raws.CommandTables[index+1:]...)
	return s.save()
}

// DropTables はドロップテーブル一覧を返す
func (s *Store) DropTables() []raw.DropTable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.DropTables
}

// DropTable は指定インデックスのドロップテーブルを返す
func (s *Store) DropTable(index int) (raw.DropTable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.DropTables) {
		return raw.DropTable{}, fmt.Errorf("ドロップテーブルインデックスが範囲外: %d", index)
	}
	return s.raws.DropTables[index], nil
}

// UpdateDropTable は指定インデックスのドロップテーブルを更新してファイルに保存する
func (s *Store) UpdateDropTable(index int, dt raw.DropTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.DropTables) {
		return fmt.Errorf("ドロップテーブルインデックスが範囲外: %d", index)
	}
	s.raws.DropTables[index] = dt
	return s.save()
}

// AddDropTable は新しいドロップテーブルを追加してファイルに保存する
func (s *Store) AddDropTable(dt raw.DropTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.DropTables = append(s.raws.DropTables, dt)
	return s.save()
}

// DeleteDropTable は指定インデックスのドロップテーブルを削除してファイルに保存する
func (s *Store) DeleteDropTable(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.DropTables) {
		return fmt.Errorf("ドロップテーブルインデックスが範囲外: %d", index)
	}
	s.raws.DropTables = append(s.raws.DropTables[:index], s.raws.DropTables[index+1:]...)
	return s.save()
}

// ItemTables はアイテムテーブル一覧を返す
func (s *Store) ItemTables() []raw.ItemTable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.ItemTables
}

// ItemTable は指定インデックスのアイテムテーブルを返す
func (s *Store) ItemTable(index int) (raw.ItemTable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.ItemTables) {
		return raw.ItemTable{}, fmt.Errorf("アイテムテーブルインデックスが範囲外: %d", index)
	}
	return s.raws.ItemTables[index], nil
}

// UpdateItemTable は指定インデックスのアイテムテーブルを更新してファイルに保存する
func (s *Store) UpdateItemTable(index int, it raw.ItemTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.ItemTables) {
		return fmt.Errorf("アイテムテーブルインデックスが範囲外: %d", index)
	}
	s.raws.ItemTables[index] = it
	return s.save()
}

// AddItemTable は新しいアイテムテーブルを追加してファイルに保存する
func (s *Store) AddItemTable(it raw.ItemTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.ItemTables = append(s.raws.ItemTables, it)
	return s.save()
}

// DeleteItemTable は指定インデックスのアイテムテーブルを削除してファイルに保存する
func (s *Store) DeleteItemTable(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.ItemTables) {
		return fmt.Errorf("アイテムテーブルインデックスが範囲外: %d", index)
	}
	s.raws.ItemTables = append(s.raws.ItemTables[:index], s.raws.ItemTables[index+1:]...)
	return s.save()
}

// EnemyTables は敵テーブル一覧を返す
func (s *Store) EnemyTables() []raw.EnemyTable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.EnemyTables
}

// EnemyTable は指定インデックスの敵テーブルを返す
func (s *Store) EnemyTable(index int) (raw.EnemyTable, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.EnemyTables) {
		return raw.EnemyTable{}, fmt.Errorf("敵テーブルインデックスが範囲外: %d", index)
	}
	return s.raws.EnemyTables[index], nil
}

// UpdateEnemyTable は指定インデックスの敵テーブルを更新してファイルに保存する
func (s *Store) UpdateEnemyTable(index int, et raw.EnemyTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.EnemyTables) {
		return fmt.Errorf("敵テーブルインデックスが範囲外: %d", index)
	}
	s.raws.EnemyTables[index] = et
	return s.save()
}

// AddEnemyTable は新しい敵テーブルを追加してファイルに保存する
func (s *Store) AddEnemyTable(et raw.EnemyTable) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.EnemyTables = append(s.raws.EnemyTables, et)
	return s.save()
}

// DeleteEnemyTable は指定インデックスの敵テーブルを削除してファイルに保存する
func (s *Store) DeleteEnemyTable(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.EnemyTables) {
		return fmt.Errorf("敵テーブルインデックスが範囲外: %d", index)
	}
	s.raws.EnemyTables = append(s.raws.EnemyTables[:index], s.raws.EnemyTables[index+1:]...)
	return s.save()
}

// Tiles はタイル一覧を返す
func (s *Store) Tiles() []raw.TileRaw {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Tiles
}

// Tile は指定インデックスのタイルを返す
func (s *Store) Tile(index int) (raw.TileRaw, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Tiles) {
		return raw.TileRaw{}, fmt.Errorf("タイルインデックスが範囲外: %d", index)
	}
	return s.raws.Tiles[index], nil
}

// UpdateTile は指定インデックスのタイルを更新してファイルに保存する
func (s *Store) UpdateTile(index int, tile raw.TileRaw) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Tiles) {
		return fmt.Errorf("タイルインデックスが範囲外: %d", index)
	}
	s.raws.Tiles[index] = tile
	return s.save()
}

// AddTile は新しいタイルを追加してファイルに保存する
func (s *Store) AddTile(tile raw.TileRaw) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Tiles = append(s.raws.Tiles, tile)
	return s.save()
}

// DeleteTile は指定インデックスのタイルを削除してファイルに保存する
func (s *Store) DeleteTile(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Tiles) {
		return fmt.Errorf("タイルインデックスが範囲外: %d", index)
	}
	s.raws.Tiles = append(s.raws.Tiles[:index], s.raws.Tiles[index+1:]...)
	return s.save()
}

// Props は置物一覧を返す
func (s *Store) Props() []raw.PropRaw {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Props
}

// Prop は指定インデックスの置物を返す
func (s *Store) Prop(index int) (raw.PropRaw, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Props) {
		return raw.PropRaw{}, fmt.Errorf("置物インデックスが範囲外: %d", index)
	}
	return s.raws.Props[index], nil
}

// UpdateProp は指定インデックスの置物を更新してファイルに保存する
func (s *Store) UpdateProp(index int, prop raw.PropRaw) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Props) {
		return fmt.Errorf("置物インデックスが範囲外: %d", index)
	}
	s.raws.Props[index] = prop
	return s.save()
}

// AddProp は新しい置物を追加してファイルに保存する
func (s *Store) AddProp(prop raw.PropRaw) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Props = append(s.raws.Props, prop)
	return s.save()
}

// DeleteProp は指定インデックスの置物を削除してファイルに保存する
func (s *Store) DeleteProp(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Props) {
		return fmt.Errorf("置物インデックスが範囲外: %d", index)
	}
	s.raws.Props = append(s.raws.Props[:index], s.raws.Props[index+1:]...)
	return s.save()
}

// Professions は職業一覧を返す
func (s *Store) Professions() []raw.Profession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.raws.Professions
}

// Profession は指定インデックスの職業を返す
func (s *Store) Profession(index int) (raw.Profession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.Professions) {
		return raw.Profession{}, fmt.Errorf("職業インデックスが範囲外: %d", index)
	}
	return s.raws.Professions[index], nil
}

// UpdateProfession は指定インデックスの職業を更新してファイルに保存する
func (s *Store) UpdateProfession(index int, prof raw.Profession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Professions) {
		return fmt.Errorf("職業インデックスが範囲外: %d", index)
	}
	s.raws.Professions[index] = prof
	return s.save()
}

// AddProfession は新しい職業を追加してファイルに保存する
func (s *Store) AddProfession(prof raw.Profession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.Professions = append(s.raws.Professions, prof)
	return s.save()
}

// DeleteProfession は指定インデックスの職業を削除してファイルに保存する
func (s *Store) DeleteProfession(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.Professions) {
		return fmt.Errorf("職業インデックスが範囲外: %d", index)
	}
	s.raws.Professions = append(s.raws.Professions[:index], s.raws.Professions[index+1:]...)
	return s.save()
}

// UpdateSpriteSheet は指定インデックスのスプライトシートを更新してファイルに保存する
func (s *Store) UpdateSpriteSheet(index int, ss raw.SpriteSheet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.SpriteSheets) {
		return fmt.Errorf("スプライトシートインデックスが範囲外: %d", index)
	}
	s.raws.SpriteSheets[index] = ss
	return s.save()
}

// SpriteSheetByIndex は指定インデックスのスプライトシートを返す
func (s *Store) SpriteSheetByIndex(index int) (raw.SpriteSheet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.raws.SpriteSheets) {
		return raw.SpriteSheet{}, fmt.Errorf("スプライトシートインデックスが範囲外: %d", index)
	}
	return s.raws.SpriteSheets[index], nil
}

// AddSpriteSheet は新しいスプライトシートを追加してファイルに保存する
func (s *Store) AddSpriteSheet(ss raw.SpriteSheet) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.raws.SpriteSheets = append(s.raws.SpriteSheets, ss)
	return s.save()
}

// DeleteSpriteSheet は指定インデックスのスプライトシートを削除してファイルに保存する
func (s *Store) DeleteSpriteSheet(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.raws.SpriteSheets) {
		return fmt.Errorf("スプライトシートインデックスが範囲外: %d", index)
	}
	s.raws.SpriteSheets = append(s.raws.SpriteSheets[:index], s.raws.SpriteSheets[index+1:]...)
	return s.save()
}
