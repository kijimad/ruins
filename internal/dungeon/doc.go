// Package dungeon はダンジョン定義と選択システムを提供する。
//
// # 責務
//
// このパッケージはダンジョンのメタデータを管理する。
// 具体的なマップ生成は mapplanner パッケージが担当し、
// このパッケージはどのダンジョンでどの種類のマップを使うかを定義する。
//
// # 主な型
//
//   - StageDefinition: ステージ種別の静的マスタ interface。名前と基本気温を共通面に持つ
//   - DungeonDefinition: フロアを生成して潜る通常ダンジョン。階層数・テーブル・プランナー抽選を持つ
//   - OverworldDefinition: 帯をスライドし続けるオーバーワールド。ダンジョン専用フィールドを持たない
//   - PlannerWeight: マップ種類と出現重みのペア
//
// # マスタとプレイ固有データの分離
//
// StageDefinition はマスタ、すなわち不変の設定であり、セーブには含めない。プレイ中の現在地は
// StageKey がキーとして持ち、ロード後は名前で GetStageDefinition から引き直す。可変でセーブ対象の
// データ、Level・探索履歴・帯状態などは StageField や SeamlessBand が別に持つ。
//
// # 使い分け
//
// ダンジョン選択画面では registry から全ダンジョンを取得して表示する。
// ダンジョン開始時は DungeonDefinition.SelectPlanner でマップ種類を抽選する。
package dungeon
