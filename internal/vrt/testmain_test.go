package vrt

import (
	"os"
	"testing"
)

// TestMain はこのパッケージのテストのエントリ。実処理は RunTestMain が持つ
func TestMain(m *testing.M) {
	os.Exit(RunTestMain(m))
}
