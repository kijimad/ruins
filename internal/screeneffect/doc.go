// Package screeneffect は画面全体に適用するポストプロセスエフェクトを提供する。
//
// 責務:
//   - ゲーム描画後の画面全体へのエフェクト適用
//   - シェーダーベースのビジュアルエフェクト管理
//   - オフスクリーンバッファの管理
//
// 使い方:
//
//	// チェーンの初期化。Filter を適用順に並べる
//	pipeline := screeneffect.NewPipeline(retro)
//
//	// フレームを描いてからチェーンを適用して画面へ出す
//	frame := layer.Begin(width, height)
//	// ... frame に描画 ...
//	pipeline.Apply(screen, frame)
//
// 仕様:
//   - Filter: 画面エフェクトを表すインターフェース
//   - Pipeline: Filter を適用順に並べたポスト処理チェーン。src へ順にかけて dst へ出す
//   - AlphaLayer: 複数の描画を平坦化してから大域アルファで一度だけ合成する層
//   - RetroFilter: 樽型歪み、色収差、ビネット、フリッカー、グロー効果を提供
//   - オフスクリーンバッファを内部で管理し、画面サイズ変更に自動対応
package screeneffect
