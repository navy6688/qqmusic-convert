package convert

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanUsesOnlyCurrentDirectoryAndPreservesDistinctFormats(t *testing.T) {
	source := t.TempDir()
	target := filepath.Join(t.TempDir(), "MP3")
	for _, name := range []string{"歌曲.mflac", "歌曲.mp3", "另一个.flac", "not-audio.txt"} {
		if err := os.WriteFile(filepath.Join(source, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	child := filepath.Join(source, "album")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "nested.mflac"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	jobs, err := Plan(source, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 3 {
		t.Fatalf("planned %d jobs, expected 3", len(jobs))
	}
	destinations := map[string]bool{}
	for _, job := range jobs {
		if destinations[job.Destination] {
			t.Fatal("two inputs share an output")
		}
		destinations[job.Destination] = true
	}
	if !destinations[filepath.Join(target, "歌曲 (mflac).mp3")] || !destinations[filepath.Join(target, "歌曲.mp3")] {
		t.Fatal("duplicate title policy is incorrect")
	}
	if _, err = os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("planning created output directory")
	}
}

func TestPlanRejectsMissingSourceAndFileDestination(t *testing.T) {
	root := t.TempDir()
	if _, err := Plan(filepath.Join(root, "missing"), filepath.Join(root, "out")); err == nil {
		t.Fatal("accepted nonexistent source")
	}
	file := filepath.Join(root, "not-directory")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Plan(root, file); err == nil {
		t.Fatal("accepted non-directory destination")
	}
}

func TestPlanCanBeRepeatedWhenExistingOutputAlsoAppearsAsSource(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"歌曲.mflac", "歌曲.mp3", "歌曲 (mflac).mp3"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := Plan(root, root)
	if err != nil {
		t.Fatalf("existing duplicate output must be skipped rather than block the whole folder: %v", err)
	}
	if len(jobs) != 3 {
		t.Fatalf("expected 3 existing-source entries, got %d", len(jobs))
	}
}
