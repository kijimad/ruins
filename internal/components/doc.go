// Package components はゲームコンポーネントの定義と実装を提供する。
//
// 責務:
// - ECS（Entity Component System）のコンポーネント定義
// - キャラクター、アイテム、フィールドオブジェクトの属性管理
// - 戦闘、移動、AI、描画などの機能別コンポーネント提供
//
// 使い分け:
// - EntitySpec: エンティティ作成時のコンポーネント情報格納
// - Components: ECSクエリで使用するコンポーネント実体
// - 各構造体: 個別のコンポーネントデータ
//
// 仕様:
// - NullComponent: 状態マーカー（Player, Dead等）
// - SliceComponent: データ保持（HP, WeightCapacity, Abilities, Attack等）
// - 死亡状態はDeadコンポーネントで明示的に管理
// - HP.Current == 0 での死亡判定とDeadコンポーネント付与を併用
//
// Interactableコンポーネント:
// - エンティティがプレイヤーと相互作用できることを示すマーカー
// - 相互作用の種類はInteractionData（Door, Talk, Item, Melee等）で定義
// - 発動範囲はActivationRange（SameTile, Adjacent）で制御
// - 発動方式はActivationWay（Auto, Manual, OnCollision）で制御
//   - Auto: 範囲内に入ったら即座に発動
//   - Manual: Enterキーやアクションメニューで発動
//   - OnCollision: 移動先として指定された時に発動（扉開閉、会話、近接攻撃等）
//
// - 環境オブジェクト（扉、アイテム）だけでなく、エンティティ（NPC、敵）も持つことができる
// - MeleeInteractionは敵が「攻撃可能な対象」であることを示す
//
// コンポーネント設計指針:
//   - インベントリのカテゴリなど恣意的な分類は、Categories定義で吸収する
//   - 「それが何であるか」を十分に絞って示す用途には、タグコンポーネントを追加する
//     例: Tile, Prop, Player はエンティティの本質的な種別を示すタグコンポーネント
//
// カテゴリシステム (invariant.go):
//   - Pred DSL (Has, And, Or, Not) でコンポーネントの組み合わせ条件を宣言的に定義する
//   - Categories() に CategoryGroupKey ごとのカテゴリ定義を集約する
//   - CategoryOf はエンティティ、CategoryOfSpec は EntitySpec に対してカテゴリを判定する
//   - カテゴリ定義はゴールデンテスト (testdata/categories.golden.json) で保護される
//   - カテゴリを追加する場合は Categories() に定義を追加し、go test -update でゴールデンファイルを更新する
//   - 順序依存がある場合はコメントとテストで明示する（例: Fire は Melee より先に判定する）
package components
