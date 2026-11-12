// Package fonts discovers font families installed on the host system.
package fonts

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/image/font/sfnt"
)

var (
	listOnce sync.Once
	cached   []string
)

var fontExtensions = map[string]bool{
	".ttf":   true,
	".otf":   true,
	".ttc":   true,
	".otc":   true,
	".dfont": true,
}

// List returns the installed font families in a stable, deduplicated order.
func List() []string {
	listOnce.Do(func() {
		cached = listFromDirs(systemDirs())
	})
	result := make([]string, len(cached))
	copy(result, cached)
	return result
}

func systemDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		return compact([]string{
			envPath("WINDIR", "Fonts"),
			envPath("LOCALAPPDATA", "Microsoft", "Windows", "Fonts"),
		})
	case "darwin":
		return compact([]string{
			"/System/Library/Fonts",
			"/Library/Fonts",
			filepath.Join(home, "Library", "Fonts"),
		})
	default:
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" && home != "" {
			dataHome = filepath.Join(home, ".local", "share")
		}
		dirs := []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			filepath.Join(home, ".fonts"),
			filepath.Join(home, ".local", "share", "fonts"),
			filepath.Join(dataHome, "fonts"),
		}
		for _, root := range filepath.SplitList(os.Getenv("XDG_DATA_DIRS")) {
			if root != "" {
				dirs = append(dirs, filepath.Join(root, "fonts"))
			}
		}
		return compact(dirs)
	}
}

func envPath(env string, rest ...string) string {
	root := os.Getenv(env)
	if root == "" {
		return ""
	}
	return filepath.Join(append([]string{root}, rest...)...)
}

func compact(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		key := strings.ToLower(filepath.Clean(value))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func listFromDirs(dirs []string) []string {
	seen := make(map[string]struct{})
	families := make([]string, 0, 64)
	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if !fontExtensions[strings.ToLower(filepath.Ext(entry.Name()))] {
				return nil
			}
			for _, family := range familiesInFile(path) {
				key := strings.ToLower(family)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				families = append(families, family)
			}
			return nil
		})
	}
	sort.SliceStable(families, func(i, j int) bool {
		left, right := strings.ToLower(families[i]), strings.ToLower(families[j])
		if left == right {
			return families[i] < families[j]
		}
		return left < right
	})
	return families
}

func familiesInFile(path string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	collection, err := sfnt.ParseCollectionReaderAt(file)
	if err != nil {
		return nil
	}

	families := make([]string, 0, collection.NumFonts())
	var buffer sfnt.Buffer
	for index := 0; index < collection.NumFonts(); index++ {
		font, err := collection.Font(index)
		if err != nil {
			continue
		}
		name, err := font.Name(&buffer, sfnt.NameIDTypographicFamily)
		if err != nil || strings.TrimSpace(name) == "" {
			name, err = font.Name(&buffer, sfnt.NameIDFamily)
		}
		if err != nil {
			continue
		}
		family := cleanFamily(name)
		if family != "" {
			families = append(families, family)
		}
	}
	return families
}

func cleanFamily(name string) string {
	name = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, name)
	name = strings.Join(strings.Fields(name), " ")
	runes := []rune(name)
	if len(runes) > 120 {
		name = strings.TrimSpace(string(runes[:120]))
	}
	return name
}
