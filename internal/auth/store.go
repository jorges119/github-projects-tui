package auth

import (
	"os"
	"strings"

	"github.com/jhermoso/ghtui/internal/config"
)

func LoadToken() (string, error) {
	path, err := config.TokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func SaveToken(token string) error {
	path, err := config.TokenPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(token)), 0600)
}

func DeleteToken() error {
	path, err := config.TokenPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
