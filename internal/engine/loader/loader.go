package loader

// ResourceLoader はすべてのリソースの読み込みを統括するインターフェース
type ResourceLoader interface {
	LoadFonts() (interface{}, error)
	LoadSpriteSheets() (interface{}, error)
	LoadRaws() (interface{}, error)
}
