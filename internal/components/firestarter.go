package components

// FireStarter は火種の道具に付くマーカー。所持していると隣接タイルの燃焼物に火をつけられる。
// 着火の可否は所持の有無だけで決まる。消費されるかどうかは FireStarter の扱う関心ではない。
type FireStarter struct{}
