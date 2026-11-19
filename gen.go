package main

// 生成はこの順に走る。後の生成はアプリをビルドして動くので、raw の型を先に書き出す。
//go:generate go run ./internal/rawschema/gen internal/rawschema
//go:generate go run . fmtraw
//go:generate go run . gencomponents
//go:generate go run . designdoc gen
//go:generate go run . genreadme
//go:generate go run . designdoc validate
