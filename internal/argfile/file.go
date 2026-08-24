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
