package genspec

// Def はコンポーネント1種の登録エントリを表す。
type Def struct {
	// Field は EntitySpec / Components 構造体のフィールド名。型名と一致させる。
	Field string
	// Comment は EntitySpec / Components フィールドに付与する行末コメント。全エントリに付ける。
	Comment string
}

// Registry は全コンポーネントの登録表。出力順もこの順序に従う。
var Registry = []Def{
	// general ================
	{Field: "Name", Comment: "表示名を保持する"},
	{Field: "Description", Comment: "説明文を保持する"},

	// item ================
	{Field: "HP", Comment: "生命力を表す。尽きると死亡する"},
	{Field: "Consumable", Comment: "一度使うと消費される消耗品を表す"},
	{Field: "WeightCapacity", Comment: "所持・格納の重量容量を表す"},
	{Field: "Melee", Comment: "近接攻撃の性能を保持する"},
	{Field: "Fire", Comment: "遠距離攻撃の性能と弾薬を保持する"},
	{Field: "Value", Comment: "アイテムの基本価値を表す"},
	{Field: "Weight", Comment: "アイテムの重量を表す"},
	{Field: "Recipe", Comment: "合成に必要な素材を保持する"},
	{Field: "Wearable", Comment: "装備品の性能を保持する"},
	{Field: "Abilities", Comment: "エンティティの能力値を保持する"},
	{Field: "Ammo", Comment: "弾薬アイテムの性能を保持する"},
	{Field: "Stackable", Comment: "スタック可能で所持数を持つことを表す"},
	{Field: "Material", Comment: "合成・売却の素材であることを示す"},
	{Field: "LocationInBackpack", Comment: "バックパック内にあることを表す"},
	{Field: "LocationEquipped", Comment: "装備中であることを表す"},
	{Field: "LocationOnField", Comment: "フィールド上にあることを表す"},
	{Field: "LocationInStorage", Comment: "収納内にあることを表す"},

	// field ================
	{Field: "Tile", Comment: "タイルエンティティであることを示す"},
	{Field: "SoloAI", Comment: "単独行動AIの設定を保持する"},
	{Field: "SquadAI", Comment: "隊員AIの設定を保持する"},
	{Field: "Camera", Comment: "カメラの位置とズームを保持する"},
	{Field: "Position", Comment: "フィールド上のピクセル座標を保持する"},
	{Field: "GridElement", Comment: "フィールド上のグリッド座標を保持する"},
	{Field: "SpriteRender", Comment: "スプライト描画情報を保持する"},
	{Field: "BlockView", Comment: "視界を遮ることを示す"},
	{Field: "BlockPass", Comment: "通行不可であることを示す"},
	{Field: "PassCost", Comment: "タイルの移動コスト修正を保持する"},
	{Field: "Door", Comment: "開閉可能な扉であることを表す"},
	{Field: "Prop", Comment: "置物であることを示す"},
	{Field: "LightSource", Comment: "光源であることを表す"},
	{Field: "Interactable", Comment: "相互作用可能であることを示す"},
	{Field: "VisualEffects", Comment: "紐づくビジュアルエフェクトを管理する"},
	{Field: "TileTemperature", Comment: "タイルの気温修正値を保持する"},

	// stage ================
	{Field: "StageBound", Comment: "束縛先ステージを保持する。往復するステージの同定に使う"},
	{Field: "StageMeta", Comment: "ステージごとのフィールド状態を保持する。現ステージは CurrentStage で引く"},
	{Field: "SeamlessBand", Comment: "オーバーワールドの帯・前線の永続状態を保持する。有無がオーバーワールド判定を兼ねる"},
	{Field: "PortalConnection", Comment: "ポータルの行き先ステージと着地座標を保持する"},
	{Field: "DungeonEntrance", Comment: "遺跡入口が進入先の遺跡定義名を保持する"},
	{Field: "Suspended", Comment: "現ステージ以外に属し稼働しないことを示すマーカー"},

	// member ================
	{Field: "Player", Comment: "操作対象の主人公であることを示す"},
	{Field: "Profession", Comment: "選択した職業を保持する"},
	{Field: "Hunger", Comment: "プレイヤーの空腹度を保持する"},
	{Field: "Wallet", Comment: "プレイヤーの資金を保持する"},
	{Field: "FactionAlly", Comment: "味方派閥であることを示す"},
	{Field: "FactionEnemy", Comment: "敵性派閥であることを示す"},
	{Field: "FactionNeutral", Comment: "中立派閥であることを示す"},
	{Field: "Boss", Comment: "ボスエンティティであることを示す"},
	{Field: "Dialog", Comment: "会話データを保持する"},
	{Field: "Dead", Comment: "死亡状態であることを示す"},
	{Field: "TurnBased", Comment: "アクションポイントを管理する"},
	{Field: "HealthStatus", Comment: "部位ごとの健康状態を保持する"},
	{Field: "Skills", Comment: "スキルセットを保持する"},
	{Field: "CharModifiers", Comment: "効果倍率を集約する"},

	// event ================
	{Field: "StateChangeRequest", Comment: "ステート遷移リクエストを運ぶ"},
	{Field: "StatsChanged", Comment: "ステータス再計算が必要なことを示すダーティフラグ"},
	{Field: "WeightDirty", Comment: "重量再計算が必要なことを示すダーティフラグ"},
	{Field: "ProvidesHealing", Comment: "HP回復の性質を保持する"},
	{Field: "ProvidesNutrition", Comment: "空腹度回復の性質を保持する"},
	{Field: "InflictsDamage", Comment: "ダメージを与える性質を保持する"},

	// book ================
	{Field: "Book", Comment: "読書可能な本であることを表す"},

	// battle ================
	{Field: "CommandTable", Comment: "AI用の戦闘コマンドテーブル名を保持する"},
	{Field: "DropTable", Comment: "ドロップテーブル名を保持する"},

	// squad ================
	{Field: "SquadMember", Comment: "隊員であることを示す"},

	// activity ================
	{Field: "Activity", Comment: "実行中のアクティビティを保持する"},
	{Field: "LastActivity", Comment: "直近のアクティビティ実行結果を保持する"},

	// singleton ================
	{Field: "GameLog", Comment: "ゲームログストレージを保持するシングルトン"},
	{Field: "Dungeon", Comment: "ダンジョン状態を保持するシングルトン"},
	{Field: "GameProgress", Comment: "ゲーム進行データを保持するシングルトン"},
	{Field: "TurnState", Comment: "ターン状態を保持するシングルトン"},
	{Field: "SpatialIndex", Comment: "空間インデックスを保持するシングルトン"},
	{Field: "WeaponSelection", Comment: "選択中の武器スロットを保持するシングルトン"},
	{Field: "GameTime", Comment: "ゲーム内時間を保持するシングルトン"},
	{Field: "VisionState", Comment: "視界計算の一時状態を保持するシングルトン"},
}
