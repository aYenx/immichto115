package photoupload

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseExtensions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]bool
	}{
		{
			name:  "basic list",
			input: "cr2,jpg,jpeg",
			want:  map[string]bool{".cr2": true, ".jpg": true, ".jpeg": true},
		},
		{
			name:  "with spaces",
			input: " cr2 , jpg , jpeg ",
			want:  map[string]bool{".cr2": true, ".jpg": true, ".jpeg": true},
		},
		{
			name:  "with dots",
			input: ".cr2,.jpg",
			want:  map[string]bool{".cr2": true, ".jpg": true},
		},
		{
			name:  "empty",
			input: "",
			want:  map[string]bool{},
		},
		{
			name:  "mixed case",
			input: "CR2,Jpg",
			want:  map[string]bool{".cr2": true, ".jpg": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExtensions(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseExtensions(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for k := range tt.want {
				if !got[k] {
					t.Fatalf("parseExtensions(%q) missing key %q", tt.input, k)
				}
			}
		})
	}
}

func TestMatchExtension(t *testing.T) {
	exts := parseExtensions("cr2,jpg,jpeg,nef")
	tests := []struct {
		name     string
		fileName string
		want     bool
	}{
		{"jpg match", "photo.jpg", true},
		{"JPG uppercase match", "photo.JPG", true},
		{"cr2 match", "photo.cr2", true},
		{"nef match", "photo.nef", true},
		{"txt no match", "readme.txt", false},
		{"no extension", "file", false},
		{"empty name", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchExtension(tt.fileName, exts)
			if got != tt.want {
				t.Fatalf("matchExtension(%q) = %v, want %v", tt.fileName, got, tt.want)
			}
		})
	}
}

func TestScan(t *testing.T) {
	// 创建临时目录结构
	tmpDir := t.TempDir()
	// 创建匹配文件
	for _, name := range []string{"photo1.jpg", "photo2.CR2", "photo3.nef"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// 创建不匹配文件
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}
	// 创建子目录中的匹配文件
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "photo4.jpeg"), []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := Scan(tmpDir, "jpg,jpeg,cr2,nef")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("Scan() returned %d entries, want 4", len(entries))
	}

	// 验证所有日期都不是零值
	for _, e := range entries {
		if e.Date.IsZero() {
			t.Errorf("entry %s has zero date", e.FileName)
		}
	}
}

func TestExtractDate_FallbackToModTime(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.cr2")
	if err := os.WriteFile(testFile, []byte("not a real RAW file"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}

	date := extractDate(testFile, info)
	// 对于没有 EXIF 的文件，应该返回文件修改时间
	diff := date.Sub(info.ModTime())
	if diff < -time.Second || diff > time.Second {
		t.Fatalf("extractDate() = %v, want close to %v", date, info.ModTime())
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"IMG_0001.CR2", "img_0001"},
		{"IMG_0001.jpg", "img_0001"},
		{"photo.JPG", "photo"},
		{"no_ext", "no_ext"},
	}
	for _, tt := range tests {
		got := baseName(tt.input)
		if got != tt.want {
			t.Errorf("baseName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestScan_RawJpgPairing(t *testing.T) {
	// RAW 文件无法解析 EXIF，但同名的 JPG 可以
	// 此测试验证 RAW 和 JPG 文件对能获得相同的日期
	tmpDir := t.TempDir()

	// 创建配对文件：RAW + JPG（均为 fake 数据,无真实 EXIF）
	// 由于都没有 EXIF，它们应该共享相同的 ModTime 回退日期
	rawFile := filepath.Join(tmpDir, "IMG_0001.CR2")
	jpgFile := filepath.Join(tmpDir, "IMG_0001.jpg")
	if err := os.WriteFile(rawFile, []byte("fake raw"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jpgFile, []byte("fake jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := Scan(tmpDir, "cr2,jpg")
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Scan() returned %d entries, want 2", len(entries))
	}

	// 找到 RAW 和 JPG 条目
	var rawEntry, jpgEntry *FileEntry
	for i := range entries {
		ext := strings.ToLower(filepath.Ext(entries[i].FileName))
		if ext == ".cr2" {
			rawEntry = &entries[i]
		} else if ext == ".jpg" {
			jpgEntry = &entries[i]
		}
	}

	if rawEntry == nil || jpgEntry == nil {
		t.Fatal("missing RAW or JPG entry")
	}

	// 两者的日期都应该是非零值
	if rawEntry.Date.IsZero() {
		t.Error("RAW entry has zero date")
	}
	if jpgEntry.Date.IsZero() {
		t.Error("JPG entry has zero date")
	}
}

