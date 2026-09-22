// icon/tray.go
package icon

import _ "embed"

// TrayPNG keeps the existing tray/menu-bar icon.
//
//go:embed albiondata-client.png
var TrayPNG []byte

// AppPNG is the Habo Client window/taskbar/executable artwork.
//
//go:embed habo-app-icon.png
var AppPNG []byte
