package genspec

// Def はコンポーネント1種の登録エントリを表す。
type Def struct {
	// Field は EntitySpec / Components 構造体のフィールド名。型名と一致させる。
	Field string
}

// Registry は全コンポーネントの登録表。出力順もこの順序に従う。
var Registry = []Def{
	// general ================
	{Field: "Name"},        // 表示名を保持する
	{Field: "RawID"},       // 生成元 raw 定義の Id を保持する同定キー
	{Field: "Description"}, // 説明文を保持する

	// item ================
	{Field: "HP"},                 // 生命力を表す。尽きると死亡する
	{Field: "Consumable"},         // 一度使うと消費される消耗品を表す
	{Field: "Material"},           // 材質を保持する。可燃性と燃焼熱量の算出根拠。不燃の材質も持つ
	{Field: "FireStarter"},        // 火種の道具のマーカー。所持で隣接の燃焼物に着火できる
	{Field: "Perishable"},         // 腐敗する食料の生成時刻と保存期間を保持する
	{Field: "WeightCapacity"},     // 所持・格納の重量容量を表す
	{Field: "Melee"},              // 近接攻撃の性能を保持する
	{Field: "Fire"},               // 遠距離攻撃の性能と弾薬を保持する
	{Field: "Value"},              // アイテムの基本価値を表す
	{Field: "Weight"},             // アイテムの重量を表す
	{Field: "Recipe"},             // クラフトに必要な素材を保持する
	{Field: "Wearable"},           // 装備品の性能を保持する
	{Field: "Abilities"},          // エンティティの能力値を保持する
	{Field: "Ammo"},               // 弾薬アイテムの性能を保持する
	{Field: "LocationInBackpack"}, // バックパック内にあることを表す
	{Field: "LocationEquipped"},   // 装備中であることを表す
	{Field: "LocationOnField"},    // フィールド上にあることを表す
	{Field: "LocationInStorage"},  // 収納内にあることを表す

	// field ================
	{Field: "Tile"},            // タイルエンティティであることを示す
	{Field: "SoloAI"},          // 単独行動AIの設定を保持する
	{Field: "Camera"},          // カメラの位置とズームを保持する
	{Field: "Position"},        // フィールド上のピクセル座標を保持する
	{Field: "GridElement"},     // フィールド上のグリッド座標を保持する
	{Field: "SpriteRender"},    // スプライト描画情報を保持する
	{Field: "BlockView"},       // 視界を遮ることを示す
	{Field: "BlockPass"},       // 通行不可であることを示す
	{Field: "PassCost"},        // タイルの移動コスト修正を保持する
	{Field: "Door"},            // 開閉可能な扉であることを表す
	{Field: "Fixed"},           // 世界に固定され拾えない固定物であることを示す
	{Field: "Pushable"},        // 押して動かせることを示す。移動拠点キューブが最初の利用者だが印は汎用
	{Field: "LightSource"},     // 光源であることを表す
	{Field: "Interactable"},    // 相互作用可能であることを示す
	{Field: "VisualEffects"},   // 紐づくビジュアルエフェクトを管理する
	{Field: "TileEnvironment"}, // タイルの環境属性。囲われ・水辺・植生
	{Field: "HeatSource"},      // 近接するキャラの低体温を回復する熱源の設定を保持する
	{Field: "Burning"},         // 燃えている状態と残りの燃焼ターン数を保持する。暖房のゲートを兼ねる

	// stage ================
	{Field: "StageBound"},       // 束縛先ステージを保持する。往復するステージの同定に使う
	{Field: "StageField"},       // ステージごとのフィールド状態を保持する。現ステージは CurrentStage で引く
	{Field: "SeamlessBand"},     // オーバーワールドの帯の永続状態を保持する。有無がオーバーワールド判定を兼ねる
	{Field: "PortalConnection"}, // ポータルの行き先ステージと着地座標を保持する
	{Field: "DungeonEntrance"},  // 遺跡入口が進入先の遺跡定義名を保持する
	{Field: "Suspended"},        // 現ステージ以外に属し稼働しないことを示すマーカー

	// member ================
	{Field: "Player"},         // 操作対象の主人公であることを示す
	{Field: "Profession"},     // 選択した職業を保持する
	{Field: "Hunger"},         // プレイヤーの空腹度を保持する
	{Field: "Wallet"},         // プレイヤーの資金を保持する
	{Field: "FactionAlly"},    // 味方派閥であることを示す
	{Field: "FactionEnemy"},   // 敵性派閥であることを示す
	{Field: "FactionNeutral"}, // 中立派閥であることを示す
	{Field: "Dialog"},         // 会話データを保持する
	{Field: "Dead"},           // 死亡状態であることを示す
	{Field: "TurnBased"},      // アクションポイントを管理する
	{Field: "HealthStatus"},   // 部位ごとの健康状態を保持する
	{Field: "Skills"},         // スキルセットを保持する

	// event ================
	{Field: "StateChangeRequest"}, // ステート遷移リクエストを運ぶ
	{Field: "StatsChanged"},       // ステータス再計算が必要なことを示すダーティフラグ
	{Field: "WeightDirty"},        // 重量再計算が必要なことを示すダーティフラグ
	{Field: "ProvidesHealing"},    // HP回復の性質を保持する
	{Field: "ProvidesNutrition"},  // 空腹度回復の性質を保持する
	{Field: "InflictsDamage"},     // ダメージを与える性質を保持する
	{Field: "Remedy"},             // 不調を治療する性質を保持する

	// book ================
	{Field: "Book"}, // 読書可能な本であることを表す

	// battle ================
	{Field: "CommandTable"}, // AI用の戦闘コマンドテーブル名を保持する
	{Field: "DropTable"},    // ドロップテーブル名を保持する

	// activity ================
	{Field: "Activity"},     // 実行中のアクティビティを保持する
	{Field: "LastActivity"}, // 直近のアクティビティ実行結果を保持する

	// singleton ================
	{Field: "GameLog"},         // ゲームログストレージを保持するシングルトン
	{Field: "Dungeon"},         // ダンジョン状態を保持するシングルトン
	{Field: "TurnState"},       // ターン状態を保持するシングルトン
	{Field: "SpatialIndex"},    // 空間インデックスを保持するシングルトン
	{Field: "WeaponSelection"}, // 選択中の武器スロットを保持するシングルトン
	{Field: "GameTime"},        // ゲーム内時間を保持するシングルトン
	{Field: "VisionState"},     // 視界計算の一時状態を保持するシングルトン
	{Field: "UserSettings"},    // 設定画面で変更するグローバル設定を保持するシングルトン
	{Field: "AuctionHistory"},  // 通信販売の金銭明細と出荷実績履歴、採番カウンタ、評判を保持するシングルトン
	{Field: "RunStats"},        // run を通じて積み上げる統計と死因を保持するシングルトン。serde 保存

	// auction ================
	{Field: "AuctionListing"}, // 通信販売で出品中の品の現在値と採番を保持する
	{Field: "AuctionSold"},    // 通信販売で落札済みで未出荷の品を示す
	{Field: "AuctionStation"}, // 通信販売の出荷場所を示すマーカー
}
