//go:build !windows

package client

func protectHaboToken(token string) (string, bool, error) {
	return token, false, nil
}

func unprotectHaboToken(value string) (string, error) {
	return value, nil
}
