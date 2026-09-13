# ローグライクの機械的バランス調整。手法調査と現行フレームワークとの比較

現行の決定論的バランス導出フレームワーク（`internal/balance` の閉形式 M(θ)・感度行列・交換レート・
マルコフDP死亡確率・凍結ゲート・Pareto前線）に対し、より良い手法や補完手法がないかを、学術・業界の
一次情報を検証して判定した記録。方針は `approach.md`、つまみ索引は `tuning.md`。

調査は deep-research で実施した。103エージェント・21一次ソース・102クレーム抽出・25検証、うち23が
反証投票を生き残った。出所種別は web は URL、inferred は本レポートの推論、と分けて記す。

## 結論

**置き換えるべき手法は、検証済み文献の範囲では見つからなかった。**（web + inferred）
学術・業界の主流手法はどれも現行の閉形式系より速度で3〜7桁遅く、収束保証がなく、現行に無い失敗モード
（退化解・エージェントの腕前バイアス・多重反復コスト）を持ち込む。一方、思想・定式化のレベルで移植
価値の高い補完が4点ある。

## 判定表

| 手法 | 何を測るか | 実測コスト | 判定 | 出典 |
|---|---|---|---|---|
| Restricted Play (Jaffe et al., AIIDE 2012) | 行動制限下の最適戦略勝率でバランスを統一定量化 | 多項式時間・決定論的 | 補完・最有力 | [washington.edu](https://homes.cs.washington.edu/~zoran/jaffe2012ecg.pdf) |
| BBExplorer (ASE 2026 preprint) | バランスが壊れる境界の発見 | 約9,000試合/パス | 定式化のみ補完 | [arXiv 2608.28364](https://arxiv.org/html/2608.28364) |
| NTBEA (Kunanusont et al., CIG 2017) | スキル階層の識別可能性 | 50試行+100再評価 | メトリクスのみ補完 | [arXiv 1705.01080](https://arxiv.org/pdf/1705.01080) |
| プレイヤー選好研究 (Pfau & Seif El-Nasr, ACM Games 2024) | symmetry拒否・viability要求の実証 | ― | 目標設定の根拠として補完 | [ACM 3675807](https://dl.acm.org/doi/abs/10.1145/3675807) |
| 進化計算バランシング (Volz et al., GECCO 2016) | シミュレーション勝率の多目的最適化 | 確率的・保証なし | 不要。Pareto探索で代替済み | [ACM 2908812](https://dl.acm.org/doi/10.1145/2908812.2908913) |
| 経済グラフ+EA (GEEvo, IEEE CEC 2024) | 経済フローのバランス | 18〜703秒/件、退化解あり | 不要 | [arXiv 2404.18574](https://arxiv.org/pdf/2404.18574) |
| AIエージェント自動プレイテスト (DRL/MCTS) | 実プレイ挙動でのバランス | DRL数日、MCTS約10時間/レベル | 日常運用は不要。仮定検証に限定 | [arXiv 2107.12061](https://arxiv.org/pdf/2107.12061), [1903.10545](https://arxiv.org/pdf/1903.10545) |

反証されて落ちたクレーム。「DRL+MCTSが人間の通過率・離脱率予測を一貫して改善する」は 0-3 で反証。
AIプレイテストの人間予測力を根拠に採用判断をしないこと。（web、[arXiv 2107.12061](https://arxiv.org/pdf/2107.12061)）

## 補完4点と、ruins への移植形

移植の具体形は inferred、すなわち検証済みクレームからの本レポートの推論。

1. **要素制限の劣化量指標**（Restricted Play型）。Jaffe らは行動制限を課した最適エージェントの勝率
   残存でバランスを測り、Monsters Divided で人間テスト前に約20回の設計反復を回して弱要素を検出した
   （勝率残存 13.54% vs 7.44%）。うちは対戦でないので勝率でなく、「武器カテゴリを封じたときの死亡
   確率・生存日数の劣化量」として M(θ) 上に定義する。劣化ゼロは死にコンテンツ、劣化突出は必須すぎ、
   という診断が決定論のまま出る。（原理は web、移植は inferred）
2. **凍結ゲートの境界マージン監視化**（BBExplorer型）。核心は「点の値」でなく「バランスが壊れる境界
   までの距離」を見る定式化。各つまみ θ_i について目標帯を割るまでの余白±何%を二分探索で出せば、
   「まだ緑だが崖まで2%」を検知できる。数値性能は単一プレプリントの自己評価なので定式化だけ借りる。
3. **スキル深度メトリクス**（NTBEA型）。持ち帰りはアルゴリズムでなく適応度設計。ティア順序を保ちつつ
   最小ギャップを最大化する発想を、装備・熟練度ティア間の体験メトリクス差の最小ギャップとして閉形式で
   定義する。「上位装備が実感できるほど強いか」を1つの数で監視する。
4. **目標帯は等強度でなくviabilityの帯**（Pfau型）。68万件調査はプレイヤーが全ビルド等強度(symmetry)を
   明確に拒否し viability を求めると示した。ビルド別ブレ幅の目標を「ゼロに寄せる」でなく「全ビルドが
   目標帯内、ただし差は残す」と張る根拠。MMOエンドゲーム由来で、シングルプレイヤーへの一般化は論文側も
   未実証。

## なぜ現行の閉形式アプローチが妥当か

- 検証済み文献で現行と同じ設計哲学、すなわち最適・厳密計算で腕前バイアスを構造的に排除する立場に立つのは
  Jaffe の Restricted Play だけ。2012年の論文が今も孤立峰なのは、学界の主流が「シミュレーションを速く
  する」方向へ行き「シミュレーションを不要にする」方向が手薄なことを示す。閉形式は傍流だが、劣るのでなく、
  ゲームごとに式を書く人的コストを払える開発体制が稀だから。内製・単一タイトルのうちはこのコストが最小。（inferred）
- GEEvo の退化解、すなわち最適化器が通常攻撃とクリティカルのダメージを等値化して数値目標を満たし、クリット
  機構の設計意図を消した事例は、目的関数だけ渡す自動調整の本質的リスクを示す。現行の「感度・交換レートを
  人間に見せて人間が決める」設計はこの失敗モードを構造的に持たない。（web、[arXiv 2404.18574](https://arxiv.org/pdf/2404.18574)）

## 調査の限界

依頼観点のうち次は検証を通過した一次情報がゼロで、判定不能のギャップとして残る。「証拠がない」のであって
「存在しない」ではない。

- (4) 商用テレメトリ実務（Slay the Spire・Hades 等のポストモーテム）
- (5) PRISM 等の確率モデル検査のバランス応用
- (6) OSSローグライク（CDDA/DCSS/Brogue/Angband/Caves of Qud）の実バランス調整プロセス
- (8) Brogue の設計哲学・fairness 議論

特に (6) は現行フレームワークと最も条件が近い比較対象なので追加調査の価値が高い。Web検索でなく、
CDDA/DCSS のリポジトリ実コードを直接読む調査が有効。

## 出典一覧（一次情報）

- Jaffe et al., "Evaluating Competitive Game Balance with Restricted Play", AIIDE 2012. https://homes.cs.washington.edu/~zoran/jaffe2012ecg.pdf
- Volz, Rudolph & Naujoks, "Demonstrating the Feasibility of Automatic Game Balancing", GECCO 2016. https://dl.acm.org/doi/10.1145/2908812.2908913 / https://arxiv.org/abs/1603.03795
- GEEvo, IEEE CEC 2024. https://arxiv.org/pdf/2404.18574
- Kunanusont et al., "The N-Tuple Bandit Evolutionary Algorithm for Game Agent Optimisation", CIG 2017. https://arxiv.org/pdf/1705.01080
- BBExplorer, ASE 2026 preprint. https://arxiv.org/html/2608.28364
- Roohi et al., CHI PLAY 2021. https://arxiv.org/pdf/2107.12061
- Zhao et al. (EA), "Winning Isn't Everything", IEEE ToG 2020. https://arxiv.org/pdf/1903.10545
- Pfau & Seif El-Nasr, ACM Games 2024. https://dl.acm.org/doi/abs/10.1145/3675807 / https://arxiv.org/pdf/2308.07576
