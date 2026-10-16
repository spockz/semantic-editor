// Package astedit exposes the body replacement preflight without preparing or publishing an edit.
package astedit

import (
	"context"
	"path/filepath"
)

// ValidateBodyTarget uses the same exact target and body-presence checks as ReplaceBody.
func ValidateBodyTarget(ctx context.Context, file, selector string) error {
	_, err := readBodyTarget(ctx, filepath.Clean(file), file, selector)
	return err
}
