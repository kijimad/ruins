// Package screeneffect は画面全体に適用するポストプロセスエフェクトを提供する。
//
// 責務:
//   - ゲーム描画後の画面全体へのエフェクト適用
//   - シェーダーベースのビジュアルエフェクト管理
//   - オフスクリーンバッファの管理
//
// 使い方:
//
//	// パイプラインの初期化
//	pipeline := screeneffect.NewPipeline(screeneffect.NewRetroFilter())
//
//	// 描画ループ内で使用
//	offscreen := pipeline.Begin(width, height)
//	// ... offscreenに描画 ...
//	pipeline.End(screen) // フィルタを適用して画面に描画
//
// 仕様:
//   - Filter: 画面エフェクトを表すインターフェース
//   - Pipeline: Filterとオフスクリーンバッファを管理する
//   - RetroFilter: 樽型歪み、色収差、ビネット、フリッカー、グロー効果を提供
//   - オフスクリーンバッファを内部で管理し、画面サイズ変更に自動対応
package screeneffect
