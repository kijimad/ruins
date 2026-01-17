// Package logger はゲーム内の構造化ログを提供する
//
// 機能別にカテゴリを分けて、デバッグしやすいログ出力を実現する。
// configパッケージからログレベル設定を受け取り、カテゴリ別の出力制御が可能。
//
// 使用例:
//
//	var log = logger.New(logger.CategoryBattle)
//	log.Info("戦闘開始", "enemy", "スライム", "hp", 100)
//	log.Debug("ダメージ計算", "base", 10, "bonus", 5)
//
// 設定:
//   - ログレベル: configパッケージのRUINS_LOG_LEVELから取得
//   - カテゴリ別ログレベル: configパッケージのRUINS_LOG_CATEGORIESから取得 (例: "battle=debug,render=warn")
package logger
