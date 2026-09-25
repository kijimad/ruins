package vrt

import (
	"os"
	"testing"
)

// TestMain はebitenループ内で全テストを走らせ、readScreen等のebiten.NewImage操作をテストで使えるようにする
func TestMain(m *testing.M) {
	os.Exit(RunTestMain(m))
}
