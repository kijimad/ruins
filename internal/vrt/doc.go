// Package vrt はビジュアルリグレッションテストの共通基盤を提供する。
//
// 公開APIは役割で5群に分かれる。
//
// # ホスト・リソース
//
//   - RunTestMain: TestMain から呼ぶ。ebiten ループ内で全テストを走らせ ebiten.NewImage 等を使えるようにする
//
// # ワールド・ステート構築
//
//   - InitReplayWorld: フルゲームを構築する重い world。states の golden_replay がフルフレームを駆動するときだけに使う。
//     widget や画面の描画テストは testutil.InitTestWorld(t, testutil.WithUI()) の軽い world を使う
//   - SetupStateMachine: ステートを構築しレイアウト確定までフレームを回す
//
// # ゴールデン比較。ピクセル差分で pass/fail を判定する
//
//   - AssertScreenGolden: 任意の画面描画を固定する
//   - DrawFrame: vrt が作った描き先に描いた画を返す。Frame.AssertGolden で名前を付けて固定する。
//     描き先は外から受け取らない。再生ドライバが撮るフレームもこの Frame で渡る
//
// # 画像生成。比較はせず保存用のPNGを返す
//
//   - RenderPNG: ステートを構築し本番の renderer で描いて、PNGを返す
//
// # 補助
//
//   - States: ステート列をビルダー関数へ変換するアダプタ
package vrt
