package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type entry struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func main() {
	ffmpeg := flag.String("ffmpeg", "", "absolute FFmpeg path")
	ffprobe := flag.String("ffprobe", "", "absolute FFprobe path")
	output := flag.String("output", "internal/media/assets", "generated asset directory")
	flag.Parse()
	if *ffmpeg == "" || *ffprobe == "" {
		fmt.Fprintln(os.Stderr, "ffmpeg and ffprobe are required")
		os.Exit(1)
	}
	if err := os.MkdirAll(*output, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var entries []entry
	for _, source := range []string{*ffmpeg, *ffprobe} {
		packed, err := pack(source, *output)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		entries = append(entries, packed)
		fmt.Printf("packed %s (%d bytes)\n", packed.Name, packed.Size)
	}
	manifest, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err = os.WriteFile(filepath.Join(*output, "manifest.json"), manifest, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func pack(source, output string) (entry, error) {
	input, err := os.Open(source)
	if err != nil {
		return entry{}, err
	}
	defer input.Close()
	name := strings.ToLower(filepath.Base(source))
	if name != "ffmpeg.exe" && name != "ffprobe.exe" {
		return entry{}, fmt.Errorf("unexpected media binary %q", name)
	}
	destination, err := os.Create(filepath.Join(output, name+".gz"))
	if err != nil {
		return entry{}, err
	}
	writer, err := gzip.NewWriterLevel(destination, gzip.BestSpeed)
	if err != nil {
		destination.Close()
		return entry{}, err
	}
	digest := sha256.New()
	count, copyErr := io.Copy(writer, io.TeeReader(input, digest))
	zipErr := writer.Close()
	closeErr := destination.Close()
	if copyErr != nil {
		return entry{}, copyErr
	}
	if zipErr != nil {
		return entry{}, zipErr
	}
	if closeErr != nil {
		return entry{}, closeErr
	}
	return entry{Name: name, SHA256: hex.EncodeToString(digest.Sum(nil)), Size: count}, nil
}
