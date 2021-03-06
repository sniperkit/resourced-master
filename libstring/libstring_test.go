package libstring

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestExpandTildeAndEnv(t *testing.T) {
	toBeTested := ExpandTildeAndEnv("~/resourced")

	if runtime.GOOS == "darwin" {
		if !strings.HasPrefix(toBeTested, "/Users") {
			t.Errorf("~ is not expanded correctly. Path: %v", toBeTested)
		}
	}

	toBeTested = ExpandTildeAndEnv("$GOPATH/src/github.com/resourced/resourced/tests/script-reader/darwin-memory.py")
	gopath := os.Getenv("GOPATH")

	if !strings.HasPrefix(toBeTested, gopath) {
		t.Errorf("$GOPATH is not expanded correctly. Path: %v", toBeTested)
	}
}

func TestGeneratePassword(t *testing.T) {
	_, err := GeneratePassword(8)
	if err != nil {
		t.Errorf("Generating password should not fail. err: %v", err)
	}
}

func TestStripChars(t *testing.T) {
	output := StripChars("She was a soul stripper. She took my heart!", "aei")
	if output != "Sh ws  soul strppr. Sh took my hrt!" {
		t.Error("StripChars did not do its job.")
	}
}
