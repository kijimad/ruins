package main

// Def はコンポーネント1種の登録エントリを表す。
type Def struct {
	// Field は EntitySpec / Components 構造体でのフィールド名。
	Field string
	// Type は ecs.Map[T] の T にあたるコンポーネント型名。Field と異なることがある。
	Type string
	// Comment は EntitySpec / Components フィールドに付与する行末コメント。全エントリに付ける。
	Comment string
}

// Registry は全コンポーネントの登録表。出力順もこの順序に従う。
var Registry = []Def{
	// general ================
	{Field: "Name", Type: "Name", Comment: "表示名を保持する"},
	{Field: "Description", Type: "Description", Comment: "説明文を保持する"},

	// item ================
	{Field: "HP", Type: "HP", Comment: "生命力を表す。尽きると死亡する"},
	{Field: "Consumable", Type: "Consumable", Comment: "一度使うと消費される消耗品を表す"},
	{Field: "WeightCapacity", Type: "WeightCapacity", Comment: "所持・格納の重量容量を表す"},
	{Field: "Melee", Type: "Melee", Comment: "近接攻撃の性能を保持する"},
	{Field: "Fire", Type: "Fire", Comment: "遠距離攻撃の性能と弾薬を保持する"},
	{Field: "Value", Type: "Value", Comment: "アイテムの基本価値を表す"},
	{Field: "Weight", Type: "Weight", Comment: "アイテムの重量を表す"},
	{Field: "Recipe", Type: "Recipe", Comment: "合成に必要な素材を保持する"},
	{Field: "Wearable", Type: "Wearable", Comment: "装備品の性能を保持する"},
	{Field: "Abilities", Type: "Abilities", Comment: "エンティティの能力値を保持する"},
	{Field: "Ammo", Type: "Ammo", Comment: "弾薬アイテムの性能を保持する"},
	{Field: "Stackable", Type: "Stackable", Comment: "スタック可能で所持数を持つことを表す"},
	{Field: "Material", Type: "Material", Comment: "合成・売却の素材であることを示す"},
	{Field: "LocationInBackpack", Type: "LocationInBackpack", Comment: "バックパック内にあることを表す"},
	{Field: "LocationEquipped", Type: "LocationEquipped", Comment: "装備中であることを表す"},
	{Field: "LocationOnField", Type: "LocationOnField", Comment: "フィールド上にあることを表す"},
	{Field: "LocationInStorage", Type: "LocationInStorage", Comment: "収納内にあることを表す"},

	// field ================
	{Field: "Tile", Type: "Tile", Comment: "タイルエンティティであることを示す"},
	{Field: "SoloAI", Type: "SoloAI", Comment: "単独行動AIの設定を保持する"},
	{Field: "SquadAI", Type: "SquadAI", Comment: "隊員AIの設定を保持する"},
	{Field: "Camera", Type: "Camera", Comment: "カメラの位置とズームを保持する"},
	{Field: "Position", Type: "Position", Comment: "フィールド上のピクセル座標を保持する"},
	{Field: "GridElement", Type: "GridElement", Comment: "フィールド上のグリッド座標を保持する"},
	{Field: "SpriteRender", Type: "SpriteRender", Comment: "スプライト描画情報を保持する"},
	{Field: "BlockView", Type: "BlockView", Comment: "視界を遮ることを示す"},
	{Field: "BlockPass", Type: "BlockPass", Comment: "通行不可であることを示す"},
	{Field: "PassCost", Type: "PassCost", Comment: "タイルの移動コスト修正を保持する"},
	{Field: "Door", Type: "Door", Comment: "開閉可能な扉であることを表す"},
	{Field: "Prop", Type: "Prop", Comment: "置物であることを示す"},
	{Field: "LightSource", Type: "LightSource", Comment: "光源であることを表す"},
	{Field: "Interactable", Type: "Interactable", Comment: "相互作用可能であることを示す"},
	{Field: "VisualEffect", Type: "VisualEffects", Comment: "紐づくビジュアルエフェクトを管理する"},
	{Field: "TileTemperature", Type: "TileTemperature", Comment: "タイルの気温修正値を保持する"},

	// member ================
	{Field: "Player", Type: "Player", Comment: "操作対象の主人公であることを示す"},
	{Field: "Profession", Type: "Profession", Comment: "選択した職業を保持する"},
	{Field: "Hunger", Type: "Hunger", Comment: "プレイヤーの空腹度を保持する"},
	{Field: "Wallet", Type: "Wallet", Comment: "プレイヤーの資金を保持する"},
	{Field: "FactionAlly", Type: "FactionAllyData", Comment: "味方派閥であることを示す"},
	{Field: "FactionEnemy", Type: "FactionEnemyData", Comment: "敵性派閥であることを示す"},
	{Field: "FactionNeutral", Type: "FactionNeutralData", Comment: "中立派閥であることを示す"},
	{Field: "Boss", Type: "Boss", Comment: "ボスエンティティであることを示す"},
	{Field: "Dialog", Type: "Dialog", Comment: "会話データを保持する"},
	{Field: "Dead", Type: "Dead", Comment: "死亡状態であることを示す"},
	{Field: "TurnBased", Type: "TurnBased", Comment: "アクションポイントを管理する"},
	{Field: "HealthStatus", Type: "HealthStatus", Comment: "部位ごとの健康状態を保持する"},
	{Field: "Skills", Type: "Skills", Comment: "スキルセットを保持する"},
	{Field: "CharModifiers", Type: "CharModifiers", Comment: "効果倍率を集約する"},

	// event ================
	{Field: "StateChangeRequest", Type: "StateChangeRequest", Comment: "ステート遷移リクエストを運ぶ"},
	{Field: "StatsChanged", Type: "StatsChanged", Comment: "ステータス再計算が必要なことを示すダーティフラグ"},
	{Field: "WeightDirty", Type: "WeightDirty", Comment: "重量再計算が必要なことを示すダーティフラグ"},
	{Field: "ProvidesHealing", Type: "ProvidesHealing", Comment: "HP回復の性質を保持する"},
	{Field: "ProvidesNutrition", Type: "ProvidesNutrition", Comment: "空腹度回復の性質を保持する"},
	{Field: "InflictsDamage", Type: "InflictsDamage", Comment: "ダメージを与える性質を保持する"},

	// book ================
	{Field: "Book", Type: "Book", Comment: "読書可能な本であることを表す"},

	// battle ================
	{Field: "CommandTable", Type: "CommandTable", Comment: "AI用の戦闘コマンドテーブル名を保持する"},
	{Field: "DropTable", Type: "DropTable", Comment: "ドロップテーブル名を保持する"},

	// squad ================
	{Field: "SquadMember", Type: "SquadMember", Comment: "隊員であることを示す"},

	// activity ================
	{Field: "Activity", Type: "Activity", Comment: "実行中のアクティビティを保持する"},
	{Field: "LastActivity", Type: "LastActivity", Comment: "直近のアクティビティ実行結果を保持する"},

	// singleton ================
	{Field: "GameLog", Type: "GameLog", Comment: "ゲームログストレージを保持するシングルトン"},
	{Field: "DungeonState", Type: "Dungeon", Comment: "ダンジョン状態を保持するシングルトン"},
	{Field: "GameProgress", Type: "GameProgress", Comment: "ゲーム進行データを保持するシングルトン"},
	{Field: "TurnState", Type: "TurnState", Comment: "ターン状態を保持するシングルトン"},
	{Field: "SpatialIndex", Type: "SpatialIndex", Comment: "空間インデックスを保持するシングルトン"},
}
