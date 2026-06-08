/*
Package save は安定ID + 静的型ベースのECSシリアライゼーションシステムを提供する。

## 概要

ECSワールドの状態をJSON形式でセーブ・ロードする。
OpenAPIスキーマ (TypeSpec → oapi-codegen) で生成された型を使い、
コンパイル時に型安全性を保証する。

## 主要な特徴

### 1. 安定ID システム
  - エンティティIDは世代管理により安定性を保証する
  - エンティティの削除・再作成時も参照の整合性を維持する
  - セーブファイル間でのエンティティ参照が安全になる

### 2. OpenAPI生成型による静的シリアライゼーション
  - oas/typespec/save.tsp で定義されたスキーマから Go 型を自動生成する
  - gc型 ↔ SaveData型の明示的な変換関数で型安全に変換する
  - ランタイムバリデーション (ValidateSaveJSON) でデータ整合性を検証する

### 3. エンティティ参照の解決
  - LocationEquipped.Owner のエンティティ参照を StableID 経由で処理する
  - セーブ時は安定IDに変換、ロード時は実エンティティに復元する

## 新しいコンポーネントの追加手順

1. oas/typespec/save.tsp にコンポーネントモデルを追加する
2. TypeSpecコンパイル → oapi-codegen で Go 型を再生成する
3. converter.go に変換関数ペアを追加する
4. manager.go の extractEntity と restoreWorldData に変換処理を追加する

## パッケージ責務

  - stable_id.go: 安定ID管理システム
  - converter.go: gc型 ↔ SaveData生成型の変換関数
  - manager.go: セーブ・ロード処理の統合管理
  - validate.go: OpenAPIスキーマによるバリデーション
*/
package save
