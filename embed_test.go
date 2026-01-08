package quickjswasi

import (
	"testing"
)

func TestEmbedWASM(t *testing.T) {
	if len(QuickJSWASM) == 0 {
		t.Fatal("QuickJSWASM is empty")
	}

	// Check for WASM magic number (0x00 0x61 0x73 0x6D)
	if len(QuickJSWASM) < 4 {
		t.Fatal("QuickJSWASM is too short to be a valid WASM file")
	}

	expected := []byte{0x00, 0x61, 0x73, 0x6D}
	for i, b := range expected {
		if QuickJSWASM[i] != b {
			t.Fatalf("Invalid WASM magic number at position %d: got 0x%02x, want 0x%02x", i, QuickJSWASM[i], b)
		}
	}

	t.Logf("Embedded WASM file size: %d bytes", len(QuickJSWASM))
}

func TestVersionInfo(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}

	t.Logf("Version: %s", Version)
	if DownloadURL != "" {
		t.Logf("DownloadURL: %s", DownloadURL)
	}
}
