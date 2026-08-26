// Package vrt はビジュアルリグレッションテストの共通基盤を提供する。
//
// 公開APIは役割で5群に分かれる。
//
// # ホスト・リソース
//
//   - RunTestMain: TestMain から呼ぶ。ebiten ループ内で全テストを走らせ ebiten.NewImage 等を使えるようにする
//   - LoadTestUIResources: テストごとに独立したUIリソースを読み込んで返す。フェイスを共有しないので並列に描ける
//
// # ワールド・ステート構築
//
//   - InitVRTWorld: 固定シードの素の決定的ワールドを作る。画面のポスト処理は切って撮る
//   - SetupStateMachine: ステートを構築しレイアウト確定までフレームを回す
//
// # ゴールデン比較。ピクセル差分で pass/fail を判定する
//
//   - AssertScreenGolden: 任意の画面描画を固定する
//   - AssertFrameGolden: 描画済みの screen を固定する。再生ドライバが撮ったフレームに使う。screen は解放しない
//
// # 画像生成。比較はせず保存用のPNGを返す
//
//   - RenderPNG: ステートを構築し本番の renderer で描いて、PNGを返す
//
// # 補助
//
//   - States: ステート列をビルダー関数へ変換するアダプタ
package vrt
