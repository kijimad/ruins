package inputmapper

// ActionID はアクションの一意な識別子
type ActionID string

// Source は1フレームぶんの入力を Action として返す供給源。第2返り値が偽なら入力なし。
// 本番はキーボードから変換した Action を返し、再生ドライバは Action 列をそのまま返す。
// キー入力を経由せずに Action を供給するための、入力層の唯一の差し替え点になる
type Source func() (ActionID, bool)

// 移動系アクション。移動は4方向のみで、斜めへは視点を回してから直進する
const (
	ActionMoveNorth ActionID = "move_north"
	ActionMoveSouth ActionID = "move_south"
	ActionMoveEast  ActionID = "move_east"
	ActionMoveWest  ActionID = "move_west"
	ActionWait      ActionID = "wait"
)

// 視点操作アクション。カメラの向きを45度単位で回す
const (
	ActionRotateLeft  ActionID = "rotate_left"
	ActionRotateRight ActionID = "rotate_right"
)

// UI系アクション
const (
	ActionOpenInventory       ActionID = "open_inventory"
	ActionOpenEquipment       ActionID = "open_equipment"
	ActionOpenCraft           ActionID = "open_craft"
	ActionOpenShop            ActionID = "open_shop"
	ActionOpenDungeonMenu     ActionID = "open_dungeon_menu"
	ActionOpenDebugMenu       ActionID = "open_debug_menu"
	ActionOpenInteractionMenu ActionID = "open_interaction_menu"
	ActionOpenFieldInfo       ActionID = "open_field_info"
	ActionOpenOverworldMap    ActionID = "open_overworld_map"
	// ActionOpenKeyHelp は現在の文脈のキー一覧ヘルプを開く
	ActionOpenKeyHelp ActionID = "open_key_help"
	ActionCloseMenu   ActionID = "close_menu"
)

// アイテム系アクション
const (
	ActionPickup  ActionID = "pickup"
	ActionDrop    ActionID = "drop"
	ActionUseItem ActionID = "use_item"
	ActionEquip   ActionID = "equip"
	ActionUnequip ActionID = "unequip"
)

// 動詞タブ画面を対応タブで開くアクション。ダンジョン・各メニューから直達する
const (
	ActionVerbExamine ActionID = "verb_examine" // 調べる
	ActionVerbPlace   ActionID = "verb_place"   // 置く
	ActionVerbConsume ActionID = "verb_consume" // 食べる。飲み物も含む
	ActionVerbRead    ActionID = "verb_read"    // 読む
	ActionVerbUse     ActionID = "verb_use"     // 使う
	ActionVerbThrow   ActionID = "verb_throw"   // 投げる
	ActionVerbList    ActionID = "verb_list"    // 出品する。タグを貼って競売にかける
)

// ActionOpenItemDetail は動詞タブ画面で選択中アイテムの詳細モーダルを開く
const ActionOpenItemDetail ActionID = "open_item_detail"

// 世界との相互作用アクション
const (
	ActionInteract ActionID = "interact" // 汎用的な相互作用（ワープ、アイテム拾得など）
)

// 戦闘系アクション
const (
	ActionAttack            ActionID = "attack"
	ActionShoot             ActionID = "shoot"
	ActionReload            ActionID = "reload"
	ActionSwitchWeaponSlot1 ActionID = "switch_weapon_slot_1"
	ActionSwitchWeaponSlot2 ActionID = "switch_weapon_slot_2"
	ActionSwitchWeaponSlot3 ActionID = "switch_weapon_slot_3"
	ActionSwitchWeaponSlot4 ActionID = "switch_weapon_slot_4"
	ActionSwitchWeaponSlot5 ActionID = "switch_weapon_slot_5"
)

// メニュー操作アクション
const (
	ActionMenuUp      ActionID = "menu_up"
	ActionMenuDown    ActionID = "menu_down"
	ActionMenuLeft    ActionID = "menu_left"
	ActionMenuRight   ActionID = "menu_right"
	ActionMenuSelect  ActionID = "menu_select"
	ActionMenuCancel  ActionID = "menu_cancel"
	ActionMenuTabNext ActionID = "menu_tab_next"
	ActionMenuTabPrev ActionID = "menu_tab_prev"
	// 対象切替。画面内で複数の対象を前後に切り替える汎用アクション
	ActionMenuSubjectPrev ActionID = "menu_subject_prev"
	ActionMenuSubjectNext ActionID = "menu_subject_next"
)

// メッセージウィンドウ系アクション
const (
	ActionConfirm ActionID = "confirm" // メッセージ確認
	ActionSkip    ActionID = "skip"    // メッセージスキップ
)
