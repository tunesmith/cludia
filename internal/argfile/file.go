// SPDX-FileCopyrightText: 2026 KeenWorks
// SPDX-License-Identifier: GPL-3.0-or-later

package argfile

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/tunesmith/cludia/internal/argument"
)

func Load(path string) ParseResult {
	return ParseFile(path)
}

func SaveAtomic(path string, doc *argument.Document) error {
	serialized, err := Serialize(doc)
	if err != nil {
		return err
	}

	directory := filepath.Dir(path)
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return statErr
	}

	temporary, err := os.CreateTemp(directory, ".cludia-*.arg")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.WriteString(serialized); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}

	dir, err := os.Open(directory)
	if err != nil {
		return nil
	}
	defer dir.Close()
	return dir.Sync()
}

// CreateAtomic creates a new file without exposing partial contents or
// overwriting an existing path. The temporary file and destination are in the
// same directory so the final hard link is atomic on the local filesystem.
func CreateAtomic(path string, doc *argument.Document) error {
	serialized, err := Serialize(doc)
	if err != nil {
		return err
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".cludia-*.arg")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.WriteString(serialized); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Link(temporaryName, path); err != nil {
		return err
	}
	dir, err := os.Open(directory)
	if err != nil {
		return nil
	}
	defer dir.Close()
	return dir.Sync()
}
