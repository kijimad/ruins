// Package uicore は UI プリミティブの実体。保持型ツリーの Widget と、その配置・描画を提供する。
//
// # 層の位置と可視性
//
// ここは Atomic Design の atom にあたる。import できるパッケージは depguard の
// ui_core_inbound_guard が列挙し、画面が名指しできるシンボルは lintrule の
// TestScreenLayerUICoreSurface が許可制で絞る。
//
// 面は2つに分ける。配置もできる Widget は部品が扱い、描くだけの Drawable を画面へ見せる。
// 画面に Layout を見せると絶対座標で画面を組めてしまい、レイアウトエンジンを迂回する経路が
// 型に開く。置き場所を決めるのは部品の仕事にする。
//
// # 責務
//
//   - 状態はインスタンスが所有する。パッケージレベルの可変状態を持たないので、
//     複数の UI を並行に構築・更新しても競合しない
//   - 配置の計算は furex の flexbox へ委譲する。Container と FlexColumn が唯一の接続点で、
//     depguard がレイアウトエンジンの import をこのパッケージだけに制限する
//   - 行分割 WrapText は UAX#14 の segmenter へ委譲し、日本語も英語も同じ規則で折り返す
//   - 描画は Canvas の裏に閉じる。テストは記録用の実装で ebiten 無しに
//     レイアウトとテキストを検証できる
package uicore
