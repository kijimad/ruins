// Package main はRuinsゲームのエントリーポイント
package main

import (
	"log"
	"os"

	_ "net/http/pprof"

	"github.com/kijimaD/ruins/internal/cmd"
)

func main() {
	app := cmd.NewMainApp()
	err := cmd.RunMainApp(app, os.Args...)
	if err != nil {
		log.Fatal(err)
	}
}
