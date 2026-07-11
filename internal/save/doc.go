/*
Package save は ark-serde によるECSワールドのセーブ・ロードを提供する。

## 概要

ark-serde でワールド全体をJSON化して保存し、ロード時にそのまま復元する（丸ごと保存）。
以前の「プレイヤー/隊員/インベントリのみ保存し、ロード時にダンジョンを再生成する」
選択的保存から、保存時の全状態を復元する方式へ移行した。

## 保存形式

セーブファイルは saveEnvelope（外枠）に ark-serde のワールドJSONを包んだ構造を持つ。

  - Version:    セーブ形式バージョン
  - Timestamp:  保存日時
  - Checksum:   破損検知用のキーレスSHA-256（Checksum自身を除いた封筒に対して計算）。
    キーを持たないため攻撃者による改ざん検知は目的とせず、あくまで破損検知に用いる
  - PlayerName: 一覧表示用にプレイヤー名をメタとして保持（ワールド全体を展開せず参照できる）
  - World:      ark-serde が出力するワールドJSON

## シリアライズの方針

  - エンティティ参照（コンポーネント内の ecs.Entity）は ark-serde が自動で再マッピングする。
    以前の手書き安定ID再マップは不要になった。
  - 一時状態コンポーネント（SpatialIndex・視界・各種ダーティフラグ・Activity 等）は
    skipComponents() で除外する。ロード時に再生成・再構築される。
  - serde 非互換なフィールド（struct をキーにした map、ebiten実行時オプション、派生データ）は
    json:"-" で除外する。
  - ark-serde の出力はマップ反復順により非決定的だが、checksum は保存バイト列に対して
    計算・検証するため整合性は保たれる。

## ロード時の復元

Deserialize はリセット済みワールドを要求するため、RestoreWorldFromJSON は先に
world.World.Reset() を行う。復元後、reestablishSingleton がスキップした一時コンポーネント
（GameLog・SpatialIndex）を再付与し、視界マップを初期化し、Resources.SingletonEntity の
参照を張り直す。

## パッケージ責務

  - serde.go:   ark-serde ラッパー、skipリスト、封筒、シングルトン再確立
  - manager.go: セーブ・ロード処理とスロット/オートセーブ管理
  - desktop.go / wasm.go: プラットフォーム別のファイルI/O
*/
package save
