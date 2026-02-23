package ignoreutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoad_FileExists(t *testing.T) {
	dir := t.TempDir()
	content := "vendor\n# comment\n\ntemp-*  \n  \n[abc]-proj\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns := Load(dir)
	want := []string{"vendor", "temp-*", "[abc]-proj"}
	if len(patterns) != len(want) {
		t.Fatalf("got %d patterns, want %d: %v", len(patterns), len(want), patterns)
	}
	for i, p := range patterns {
		if p != want[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, p, want[i])
		}
	}
}

func TestLoad_FileNotExists(t *testing.T) {
	dir := t.TempDir()
	patterns := Load(dir)
	if patterns != nil {
		t.Fatalf("expected nil, got %v", patterns)
	}
}

func TestLoad_FileUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not effective on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root ignores file permissions")
	}

	dir := t.TempDir()
	fp := filepath.Join(dir, FileName)
	if err := os.WriteFile(fp, []byte("vendor\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(fp, 0000); err != nil {
		t.Fatal(err)
	}

	patterns := Load(dir)
	if patterns != nil {
		t.Fatalf("expected nil for unreadable file, got %v", patterns)
	}
}

func TestIsIgnored_LiteralMatch(t *testing.T) {
	patterns := []string{"vendor"}
	if !IsIgnored("vendor", patterns) {
		t.Error("expected 'vendor' to be ignored")
	}
	if IsIgnored("vendor2", patterns) {
		t.Error("expected 'vendor2' NOT to be ignored")
	}
	if IsIgnored("my-vendor", patterns) {
		t.Error("expected 'my-vendor' NOT to be ignored")
	}
}

func TestIsIgnored_GlobStar(t *testing.T) {
	patterns := []string{"temp-*"}
	if !IsIgnored("temp-foo", patterns) {
		t.Error("expected 'temp-foo' to be ignored")
	}
	if IsIgnored("vendor", patterns) {
		t.Error("expected 'vendor' NOT to be ignored")
	}
	if IsIgnored("temperature", patterns) {
		t.Error("expected 'temperature' NOT to be ignored")
	}
}

func TestIsIgnored_QuestionMark(t *testing.T) {
	patterns := []string{"fo?"}
	if !IsIgnored("foo", patterns) {
		t.Error("expected 'foo' to be ignored")
	}
	if IsIgnored("foobar", patterns) {
		t.Error("expected 'foobar' NOT to be ignored")
	}
	if IsIgnored("fo", patterns) {
		t.Error("expected 'fo' NOT to be ignored")
	}
}

func TestIsIgnored_CharClass(t *testing.T) {
	patterns := []string{"[abc]-proj"}
	if !IsIgnored("a-proj", patterns) {
		t.Error("expected 'a-proj' to be ignored")
	}
	if IsIgnored("d-proj", patterns) {
		t.Error("expected 'd-proj' NOT to be ignored")
	}
}

func TestIsIgnored_EmptyPatterns(t *testing.T) {
	if IsIgnored("anything", nil) {
		t.Error("expected nothing to be ignored with nil patterns")
	}
}

func TestIsIgnored_BadPattern(t *testing.T) {
	patterns := []string{"["}
	if IsIgnored("anything", patterns) {
		t.Error("expected bad pattern to not match")
	}
}

func TestLoad_StripsLeadingSlashes(t *testing.T) {
	dir := t.TempDir()
	content := "/random-stuff\n//double\n/\n"
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	patterns := Load(dir)
	want := []string{"random-stuff", "double"}
	if len(patterns) != len(want) {
		t.Fatalf("got %d patterns, want %d: %v", len(patterns), len(want), patterns)
	}
	for i, p := range patterns {
		if p != want[i] {
			t.Errorf("pattern[%d] = %q, want %q", i, p, want[i])
		}
	}

	// Verify the stripped pattern actually matches the directory name.
	if !IsIgnored("random-stuff", patterns) {
		t.Error("expected 'random-stuff' to be ignored after stripping leading slash")
	}
}

func TestIsIgnored_MultiplePatterns(t *testing.T) {
	patterns := []string{"vendor", "temp-*"}
	if !IsIgnored("vendor", patterns) {
		t.Error("expected 'vendor' to be ignored")
	}
	if !IsIgnored("temp-foo", patterns) {
		t.Error("expected 'temp-foo' to be ignored")
	}
	if IsIgnored("keeper", patterns) {
		t.Error("expected 'keeper' NOT to be ignored")
	}
}
