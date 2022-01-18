package tmpdir

import (
	"fmt"
	"os"
)

// Expected usage is like:
//
// myFilePath := rptmp.Dir() + "/myfile"
//
// The idea is that we'll always be putting our tmp files in the same place.
// We'll be able to find where we put everything and we'll be able to clean it
// very easily. And if we need to run in multiple processes with _multiple_
// independent tmp directories, we can change it so that `dir` is defined by an
// environment variable.

var DefaultDir = "/tmp/mytmp"

func getDir() string {
	dirFromEnv := os.Getenv("MY_TMP_DIR")
	if dirFromEnv != "" {
		return dirFromEnv
	}
	return DefaultDir
}

func Dir() string {
	if err := Make(); err != nil {
		fmt.Println("erroring making tmpdir: %w", err)
	}
	return getDir()
}

func Make() error {
	return os.MkdirAll(getDir(), os.ModePerm)
}

func Remove() error {
	return os.RemoveAll(getDir())
}

func Reset() error {
	err := Remove()
	if err != nil {
		return fmt.Errorf("removing %s: %w", getDir(), err)
	}
	err = Make()
	if err != nil {
		return fmt.Errorf("making %s: %w", getDir(), err)
	}
	return nil
}
