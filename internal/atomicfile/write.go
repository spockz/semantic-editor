// Package atomicfile centralizes atomic publication so pipelines and LSP edit application share one disk-write contract.
package atomicfile

import (
	"errors"
	"fmt"
)

// ErrAtomicWrite identifies failures during staging, syncing, or publication.
var ErrAtomicWrite = errors.New("atomic write failed")

// WriteAtomic publishes bytes with preserved permissions and an advancing modification time.
func WriteAtomic(targetPath string, data []byte) error {
	return WriteAtomicChecked(targetPath, data, nil)
}

// WriteAtomicChecked validates an expected revision after staging and immediately before publication.
func WriteAtomicChecked(targetPath string, data []byte, check func() error) error {
	staged, err := Stage(targetPath, data)
	if err != nil {
		return err
	}
	defer staged.Discard()
	if check != nil {
		if err := check(); err != nil {
			return fmt.Errorf("%w: validate target before publication: %w", ErrAtomicWrite, err)
		}
	}
	return staged.Publish()
}
