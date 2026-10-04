//go:build windows

package client

import (
	"encoding/base64"
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func protectHaboToken(token string) (string, bool, error) {
	plain := []byte(token)
	if len(plain) == 0 {
		return "", true, nil
	}

	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", true, err
	}
	if out.Data == nil || out.Size == 0 {
		return "", true, errors.New("DPAPI returned an empty protected token")
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))

	protected := make([]byte, int(out.Size))
	copy(protected, unsafe.Slice(out.Data, int(out.Size)))
	return base64.StdEncoding.EncodeToString(protected), true, nil
}

func unprotectHaboToken(value string) (string, error) {
	protected, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(protected) == 0 {
		return "", nil
	}

	in := windows.DataBlob{Size: uint32(len(protected)), Data: &protected[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", err
	}
	if out.Data == nil || out.Size == 0 {
		return "", errors.New("DPAPI returned an empty token")
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))

	plain := make([]byte, int(out.Size))
	copy(plain, unsafe.Slice(out.Data, int(out.Size)))
	return string(plain), nil
}
