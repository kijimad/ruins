---
name: ruins-code
description: ruins のコード規約。doc.go・コメント・ゲッター/コンストラクタ・YAGNI・公開API・lint・テストの書き方。Goコードを書く/直すときに使う
---

# ruins コード規約

Go のベテランとして、シンプルさとテストしやすさを最優先する。責務によって package を分け、Go のベストプラクティスに従う。

## パッケージ

- 各 package の `doc.go` に記載する: package の説明 / 使い分け / 責務 / 仕様。
- 責務によって適切に package を分ける。

## コメント

- ロジックやコメントは日本語で書く。
- 注意が必要な箇所だけを最小限コメントする。見ればわかる自明なコメントは書かない。
- コメントに体言止めを使わない（意味が不明瞭になるため）。
- コメントのカッコでの補足は最小限にし、完全な文章で説明する。
- **定常状態を書く。変更ナラティブを書かない。** 「今どうなっているか」だけを説明し、「従来は〜」「〜へ移行」「〜に再設計した」「旧〜との互換」のような旧実装との対比や変更の経緯は書かない。ブランチがマージされて main になれば「以前」は存在せず、diff の注釈が古びて残るだけになる。到達の経緯は git 履歴、意思決定は設計 doc に置く。ランタイムの一時的状態を指す「旧位置」等はこの限りでない。
- **外部ツールの知識を前提にしない。** 「CDDA の OMT に相当」「Minecraft の RandomSpread の翻案」のように、そのツールを知らないと意味が取れないコメントを書かない。仕組みはその場で自己完結させて説明する。翻案元の系譜は設計 doc に残す。

## 設計・API

- 不要なゲッター/セッター関数は定義しない。
- コンストラクタは1つの型につき最大1つ。必要ならオプションパターンで定義する。
- YAGNI 原則に従い、使わないうちから実装しない。ただし余地のある設計はしておく。
- 定義する関数は最小限に保つ。呼び出されていない関数は削除する。
- 公開 API を最小限にする。private と public を明確に区別する。
- enum は基本的に `type X string` にし、`iota` は使わない。値をそのままログや `%v` に出せて、デバッグで種別名が読める。iota の連番は保存値の互換や表示のたびに `String()` を要して割に合わない。既存の `PlannerType` / `CombatPolicy`（`internal/components/ai_policy.go`）が定石。序列だけが要る箇所は別途 order スライスを持ち、`slices.Sort` の暗黙序列に頼らない。
- linter ルールは無視設定しない。
- 生成ファイル（`*_gen.go`）は手編集しない。生成元を直して再生成する。
- import 文は手編集しない。本体コードだけ書いて `make fmt`（goimports）に増減を任せる。慣例エイリアス（`gc`/`w`/`ecs` 等）も既存ファイルから学習して付く。手で足し引きすると書き間違い・消し忘れ・エイリアス不一致の入り口を増やすだけ。生の `go tool goimports` も叩かず make ターゲットを使う。

## メニュー画面

メニュー系 state は共通ランタイム `states.Screen[Props]` の上に載せ、state と描画を分離する。設計は `docs/design/260804014116.md`。

- state 構造体に `widget *ebitenui.UI`・`rebuild bool`・`*hooks.Mount` を直接持たせない。これらは UI ランタイムで、`Screen[Props]` が保持する。state はドメイン状態だけを持つ。
- `Update` は `st.screen.Update(world, st)` へ、`Draw` は `st.screen.Draw(screen)` へ委譲する。6手順の骨格を各 state に写経しない。
- `view` は state を読まない。`view(world, props, sel, res)` の引数だけから widget を組む純粋関数にする。`st.` を読み始めたら分離できていない合図。現在値は `st.screen.Props()`・`st.screen.Selection()` から取る。
- 詳細モーダルやアクション窓は `menuscreen.Overlay` として `NewScreen` に登録する。入力ゲートと重ねは Screen が扱う。入れ子や第2カーソルの変則画面も Overlay の合成で表現し、Screen 本体に画面固有の条件分岐を足さない。`isXxxScreen` 相当のフラグが Screen に生えたら誤った抽象の警報。
- 「本文＋選択肢」の単純メニューは `states.ChoiceMenu` を使う。`messagedata` はナラティブのメッセージ提示に用途を限り、メニューには使わない。
- テキスト入力フォームなど list メニューに素直に載らない画面は無理に載せず bespoke で残す。

## テスト

- 極力 `github.com/stretchr/testify` のアサート（`assert.Error` / `require.Error` 等）を使う。`t.Error()` / `t.Fatal()` は使わない。
- テストしやすく設計し、テストを追加する。
- エンティティを使う挙動テストは本番の spawn 関数（`SpawnPlayer`/`SpawnDoor`/`SpawnEnemy`/`SpawnFieldItem` 等）で組む。`world.ECS.NewEntity()` + 個別 `Components.X.Add` で手作りすると、本番の spawn が付ける他コンポーネントを取りこぼしてテストと本番が乖離し、バグを見逃す。扉が `LocationOnField` を持つ事実を手動テストが欠き、占有判定の不具合をマージ後まで見逃した例がある。「あるコンポーネントの有無」を述語にするときは、その属性を持つ別実体が無いか確認し、正準述語（`IsPickable` 等）があれば借りる。
- 実行は `make test`（Ebiten のウィンドウが出るのを防ぐため）。
- テスト名は `Test<対象関数名>_<日本語で検証内容>` にし、名前だけで何を検証するか分かるようにする。例 `func TestNewItemSpec_本のスキル未指定はエラー(t *testing.T)`。table-driven の場合は関数名を `Test<対象関数名>` とし、各ケースを `t.Run("<日本語で検証内容>", ...)` で表す。
- アサートは検証対象の種類で選び、過不足のない厳密さにする。`require.Error` だけの、内容を見ない自明なエラーテストにはしない。
  - 値すなわち戻り値や計算結果は `assert.Equal` で全文一致させる。自分が生成する確定値ほど、厳密一致が回帰を捕まえる。
  - **どのエラーが起きたかの同定は `require.ErrorIs` / `errors.As` で行う。`assert.Contains(err.Error(), ...)` で文字列照合しない。** 文字列一致はたまたま同じ文言の別エラーでも通る脆い判定で、本番が sentinel を返さない不整合も隠す。同定したいエラーに sentinel や型が無ければ定義し、本番の生成側も sentinel を返すよう直す。`require.ErrorIs` は err の非 nil も兼ねるので `require.Error` と重ねない。
  - 例外は値を埋め込む動的な検証メッセージの中身確認だけ。この場合に限り `assert.ErrorContains` で安定した意味のある断片を見る。エラー全文は下位層・OS・標準ライブラリの文言やパス・ID が混ざって脆いので、全文一致させない。
  - 既存の `Contains(err.Error(), ...)` は触れた箇所から段階的に `ErrorIs` へ移す。一括移行はしない。
  - `assert.Contains` はコレクションの包含判定にだけ使う。文字列の部分一致を全文一致の代わりに使わない。

## 検証

- コード変更後は `make check` を実行して壊していないか検証する。
