// Package main はRuinsゲームのエントリーポイント
package main

import (
	"log"
	"os"
	"runtime/debug"

	_ "net/http/pprof"

	"github.com/kijimaD/ruins/internal/cmd"
)

//go:generate go run github.com/mikefarah/yq/v4@v4.53.6 -p toml -o toml -i assets/metadata/entities/raw/raw.toml
//go:generate go run . gencomponents
//go:generate go run . designdoc gen
//go:generate go run . genreadme
//go:generate go run . designdoc validate

func main() {
	// 起動最初期に呼ぶ。未捕捉 panic 時のトレースへ全 goroutine を含める。設定読み込みより前で
	// 落ちてもトレースが薄くならないよう、ここが最も早い
	debug.SetTraceback("all")

	app := cmd.NewMainApp()
	err := cmd.RunMainApp(app, os.Args...)
	if err != nil {
		log.Fatal(err)
	}
}
