package editor

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

// indexItem はテンプレートに渡すアイテム行データ
type indexItem struct {
	Index  int
	Item   raw.Item
	Active bool
}

// editData はアイテム編集テンプレートに渡すデータ
type editData struct {
	Index      int
	Item       raw.Item
	SheetNames []string
}

// indexData はindexテンプレートに渡すデータ
type indexData struct {
	Items []indexItem
	Edit  *editData
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	selected := -1
	if v := r.URL.Query().Get("selected"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			selected = n
		}
	}
	s.renderIndex(w, selected)
}

func (s *Server) renderIndex(w http.ResponseWriter, activeIndex int) {
	items := s.store.Items()
	rows := make([]indexItem, len(items))
	for i, item := range items {
		rows[i] = indexItem{Index: i, Item: item, Active: i == activeIndex}
	}
	data := indexData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(items) {
		data.Edit = &editData{
			Index:      activeIndex,
			Item:       items[activeIndex],
			SheetNames: s.sheetNames(),
		}
	}
	if err := s.templates.ExecuteTemplate(w, "index", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// findItemIndex は名前からアイテムのインデックスを返す。見つからなければ-1を返す
func (s *Server) findItemIndex(name string) int {
	for i, item := range s.store.Items() {
		if item.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleItemUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	item, err := s.store.Item(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	item = parseItemForm(r, item)

	if err := s.store.UpdateItem(index, item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ソートによりインデックスが変わるため、更新後の名前でインデックスを探す
	newIndex := s.findItemIndex(item.Name)
	http.Redirect(w, r, fmt.Sprintf("/items?selected=%d", newIndex), http.StatusSeeOther)
}

func (s *Server) handleItemCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}

	item := raw.Item{
		Name:        name,
		Description: r.FormValue("description"),
	}
	if err := s.store.AddItem(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newIndex := s.findItemIndex(name)
	http.Redirect(w, r, fmt.Sprintf("/items?selected=%d", newIndex), http.StatusSeeOther)
}

func (s *Server) handleItemDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteItem(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/items", http.StatusSeeOther)
}

// itemSelectOption はセレクトボックス用のアイテム選択肢
type itemSelectOption struct {
	Name  string
	Label string
}

// itemOptions はアイテムの選択肢を種別順・名前順で返す
func (s *Server) itemOptions() []itemSelectOption {
	items := s.store.Items()
	// sortItemsで種別順にソート済み（Store.load/save時にソートされている）
	opts := make([]itemSelectOption, len(items))
	for i, item := range items {
		opts[i] = itemSelectOption{
			Name:  item.Name,
			Label: itemTypeLabel(item) + item.Name,
		}
	}
	return opts
}

// itemTypeLabel はアイテムの種別をテキストラベルで返す
func itemTypeLabel(item raw.Item) string {
	var labels []string
	if item.Melee != nil {
		labels = append(labels, "近")
	}
	if item.Fire != nil {
		labels = append(labels, "射")
	}
	if item.Wearable != nil {
		labels = append(labels, "防")
	}
	if item.Consumable != nil {
		labels = append(labels, "消")
	}
	if item.Ammo != nil {
		labels = append(labels, "弾")
	}
	if item.Book != nil {
		labels = append(labels, "本")
	}
	if len(labels) == 0 {
		return ""
	}
	return "[" + strings.Join(labels, "") + "] "
}

// parseItemForm はHTTPフォームからItem構造体にフィールドを反映する
func parseItemForm(r *http.Request, item raw.Item) raw.Item {
	item.Name = r.FormValue("name")
	item.Description = r.FormValue("description")
	item.SpriteSheetName = r.FormValue("sprite_sheet_name")
	item.SpriteKey = r.FormValue("sprite_key")

	if v := r.FormValue("value"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			item.Value = n
		}
	} else {
		item.Value = 0
	}

	if v := r.FormValue("weight"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			item.Weight = &f
		}
	} else {
		item.Weight = nil
	}

	if v := r.FormValue("inflicts_damage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			item.InflictsDamage = &n
		}
	} else {
		item.InflictsDamage = nil
	}

	if v := r.FormValue("provides_nutrition"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			item.ProvidesNutrition = &n
		}
	} else {
		item.ProvidesNutrition = nil
	}

	stackable := r.FormValue("stackable") == "on"
	if stackable {
		item.Stackable = &stackable
	} else {
		item.Stackable = nil
	}

	// Melee
	if r.FormValue("has_melee") == "on" {
		if item.Melee == nil {
			item.Melee = &raw.MeleeRaw{}
		}
		item.Melee.Accuracy, _ = strconv.Atoi(r.FormValue("melee_accuracy"))
		item.Melee.Damage, _ = strconv.Atoi(r.FormValue("melee_damage"))
		item.Melee.AttackCount, _ = strconv.Atoi(r.FormValue("melee_attack_count"))
		item.Melee.Cost, _ = strconv.Atoi(r.FormValue("melee_cost"))
		item.Melee.Element = r.FormValue("melee_element")
		item.Melee.AttackCategory = r.FormValue("melee_attack_category")
		item.Melee.TargetGroup = r.FormValue("melee_target_group")
		item.Melee.TargetNum = r.FormValue("melee_target_num")

		if item.Weapon == nil {
			item.Weapon = &raw.Weapon{}
		}
	} else {
		item.Melee = nil
	}

	// Fire
	if r.FormValue("has_fire") == "on" {
		if item.Fire == nil {
			item.Fire = &raw.FireRaw{}
		}
		item.Fire.Accuracy, _ = strconv.Atoi(r.FormValue("fire_accuracy"))
		item.Fire.Damage, _ = strconv.Atoi(r.FormValue("fire_damage"))
		item.Fire.AttackCount, _ = strconv.Atoi(r.FormValue("fire_attack_count"))
		item.Fire.Cost, _ = strconv.Atoi(r.FormValue("fire_cost"))
		item.Fire.Element = r.FormValue("fire_element")
		item.Fire.AttackCategory = r.FormValue("fire_attack_category")
		item.Fire.TargetGroup = r.FormValue("fire_target_group")
		item.Fire.TargetNum = r.FormValue("fire_target_num")
		item.Fire.MagazineSize, _ = strconv.Atoi(r.FormValue("fire_magazine_size"))
		item.Fire.ReloadEffort, _ = strconv.Atoi(r.FormValue("fire_reload_effort"))
		item.Fire.AmmoTag = r.FormValue("fire_ammo_tag")

		if item.Weapon == nil {
			item.Weapon = &raw.Weapon{}
		}
	} else {
		item.Fire = nil
	}

	// Consumable
	if r.FormValue("has_consumable") == "on" {
		if item.Consumable == nil {
			item.Consumable = &raw.Consumable{}
		}
		item.Consumable.UsableScene = r.FormValue("consumable_usable_scene")
		item.Consumable.TargetGroup = r.FormValue("consumable_target_group")
		item.Consumable.TargetNum = r.FormValue("consumable_target_num")
	} else {
		item.Consumable = nil
	}

	// Wearable
	if r.FormValue("has_wearable") == "on" {
		if item.Wearable == nil {
			item.Wearable = &raw.Wearable{}
		}
		item.Wearable.Defense, _ = strconv.Atoi(r.FormValue("wearable_defense"))
		item.Wearable.EquipmentCategory = r.FormValue("wearable_equipment_category")
		item.Wearable.InsulationCold, _ = strconv.Atoi(r.FormValue("wearable_insulation_cold"))
		item.Wearable.InsulationHeat, _ = strconv.Atoi(r.FormValue("wearable_insulation_heat"))
	} else {
		item.Wearable = nil
	}

	return item
}
