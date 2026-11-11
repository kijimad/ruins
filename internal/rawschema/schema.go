package rawschema

import "embed"

//go:generate go run ./gen

// SchemaFiles は raw のスキーマの CUE のファイル。検証は rawcheck が行う。
//
//go:embed *.cue
var SchemaFiles embed.FS
