package shell

import (
	"errors"
	"fmt"
	"os"

	"oras.land/oras-go/pkg/content"
)

func Begin() (*content.OCIStore, *ShellLogin, error) {
	var (
		storeDir, loginDir string
		err                error
	)

	// Initiate directories
	val := os.Getenv("ORAS_BEGIN_ENV")
	if val != "" {
		storeDir, loginDir, err = environment()
		if err != nil {
			return nil, nil, err
		}
	} else {
		storeDir, loginDir, err = arguments()
		if err != nil {
			return nil, nil, err
		}
	}

	// Store
	store, err := content.NewOCIStore(storeDir)
	if err != nil {
		return nil, nil, Err.CouldNotOpenStore
	}

	// Load index if we have it
	err = store.LoadIndex()
	if err != nil {
		return nil, nil, Err.CouldNotLoadStoreIndex
	}

	sh := NewLogin(loginDir)
	if sh == nil {
		return nil, nil, Err.CouldNotCreateAuthLogin
	}

	return store, sh, nil
}

// environment - processes directories from env variables
func environment() (storeDir, loginDir string, err error) {
	storeDir = os.Getenv("ORAS_STORE_DIR")
	loginDir = os.Getenv("ORAS_LOGIN_DIR")

	err = validate(storeDir)
	if err != nil {
		return "", "", fmt.Errorf("env: storedir: %w", err)
	}

	err = validate(loginDir)
	if err != nil {
		return "", "", fmt.Errorf("env: logindir: %w", err)
	}

	return storeDir, loginDir, nil
}

// arguments - processes directories from cmd-line arguments
func arguments() (storeDir, loginDir string, err error) {
	args := os.Args
	if len(args) < 2 {
		return "", "", Err.NotEnoughArguments
	}

	// Arguments from command line
	storeDir = args[1]
	loginDir = args[2]

	err = validate(storeDir)
	if err != nil {
		return "", "", fmt.Errorf("args: storedir: %w", err)
	}

	err = validate(loginDir)
	if err != nil {
		return "", "", fmt.Errorf("args: logindir: %w", err)
	}

	return storeDir, loginDir, nil
}

func validate(dir string) error {
	// Process store path
	fi, err := os.Stat(dir)
	if err != nil {
		return Err.PathDoesNotExist
	}

	if !fi.IsDir() {
		return Err.NotDirectory
	}

	return nil
}

var Err = struct {
	NotEnoughArguments      error
	PathDoesNotExist        error
	NotDirectory            error
	CouldNotOpenStore       error
	CouldNotLoadStoreIndex  error
	CouldNotCreateAuthLogin error
}{
	errors.New("not enough arguments"),
	errors.New("path does not exist"),
	errors.New("not a directory"),
	errors.New("could not open store"),
	errors.New("could not load index"),
	errors.New("could not create auth login"),
}
