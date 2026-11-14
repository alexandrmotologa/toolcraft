package hardener_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/hardener"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
)

func TestMemoryFootprintUnder15MB(t *testing.T) {
	runtime.GC()
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)

	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	_, _, err = hardener.Harden(spec, config.DefaultConfig().Hardening)
	if err != nil {
		t.Fatalf("failed to harden: %v", err)
	}

	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)

	allocBytes := mAfter.Alloc - mBefore.Alloc
	allocMB := float64(allocBytes) / (1024 * 1024)

	t.Logf("Memory consumed by parse & harden: %.2f MB (Alloc: %d bytes)", allocMB, allocBytes)
	if allocMB > 15.0 {
		t.Errorf("memory consumption exceeded 15MB threshold: %.2f MB", allocMB)
	}
}
