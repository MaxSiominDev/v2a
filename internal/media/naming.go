package media

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WithExtension(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

func SuffixedName(path, suffix string) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return base + "." + suffix + ext
}

func CollisionPath(path string, n int) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s (%d)%s", base, n, ext)
}

// ResolveCollision returns path itself, or the first free "path (n)" variant
func ResolveCollision(path string) (string, error) {
	exists, err := pathExists(path)
	if err != nil {
		return "", fmt.Errorf("failed to check output path %s: %w", path, err)
	}
	if !exists {
		return path, nil
	}

	for n := 1; ; n++ {
		candidate := CollisionPath(path, n)
		exists, err := pathExists(candidate)
		if err != nil {
			return "", fmt.Errorf("failed to check output path %s: %w", candidate, err)
		}
		if !exists {
			return candidate, nil
		}
	}
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, err
	}
}
