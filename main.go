// Package main はRuinsゲームのエントリーポイント
package main

import (
	"log"
	"os"

	_ "net/http/pprof"

	"github.com/kijimaD/ruins/internal/cmd"
)

// raw.toml を yq で正規化する。yq はデータモデル型で決定論的な正規形を出すため、jq 風の式で
// フィールドを編集しても差分が変更箇所だけに収まる。taplo は map 文字列を複数行化して \r を混ぜ
// 壊すので assets 配下は .taplo.toml で除外しており、raw の整形はこの yq が単独で担う。
//go:generate go run github.com/mikefarah/yq/v4@v4.53.6 -p toml -o toml -i assets/metadata/entities/raw/raw.toml
//go:generate go run . gencomponents
//go:generate go run . designdoc gen
//go:generate go run . genreadme
//go:generate go run . designdoc validate

func main() {
	app := cmd.NewMainApp()
	err := cmd.RunMainApp(app, os.Args...)
	if err != nil {
		log.Fatal(err)
	}
}
