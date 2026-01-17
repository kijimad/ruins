package logger

// Category は機能別のログカテゴリを表す型
type Category string

const (
	// CategoryTransition はステート遷移のログカテゴリ
	CategoryTransition Category = "transition" // ステート遷移

	// CategoryMove は移動処理のログカテゴリ
	CategoryMove Category = "move" // 移動処理
	// CategoryInput は入力処理のログカテゴリ
	CategoryInput Category = "input" // 入力処理
	// CategoryRender は描画処理のログカテゴリ
	CategoryRender Category = "render" // 描画処理
	// CategoryVision は視界計算のログカテゴリ
	CategoryVision Category = "vision" // 視界計算
	// CategoryHUD はHUD表示のログカテゴリ
	CategoryHUD Category = "hud" // HUD表示

	// CategoryAction はアクション処理のログカテゴリ
	CategoryAction Category = "action" // アクション処理

	// CategoryTurn はターン管理のログカテゴリ
	CategoryTurn Category = "turn" // ターン管理

	// CategoryEffect はエフェクト全般のログカテゴリ
	CategoryEffect Category = "effect" // エフェクト全般
	// CategoryDamage はダメージ処理のログカテゴリ
	CategoryDamage Category = "damage" // ダメージ処理
	// CategoryHeal は回復処理のログカテゴリ
	CategoryHeal Category = "heal" // 回復処理

	// CategoryResource はリソース管理のログカテゴリ
	CategoryResource Category = "resource" // リソース管理
	// CategoryLoad はファイル読み込みのログカテゴリ
	CategoryLoad Category = "load" // ファイル読み込み
	// CategoryCache はキャッシュ管理のログカテゴリ
	CategoryCache Category = "cache" // キャッシュ管理

	// CategoryWorld はワールド管理のログカテゴリ
	CategoryWorld Category = "world" // ワールド管理
	// CategoryEntity はエンティティ操作のログカテゴリ
	CategoryEntity Category = "entity" // エンティティ操作
	// CategoryComponent はコンポーネント操作のログカテゴリ
	CategoryComponent Category = "component" // コンポーネント操作

	// CategoryPerf はパフォーマンス計測のログカテゴリ
	CategoryPerf Category = "perf" // パフォーマンス計測
	// CategoryDebug はデバッグ全般のログカテゴリ
	CategoryDebug Category = "debug" // デバッグ全般
)
