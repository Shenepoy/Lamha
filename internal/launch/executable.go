// Package launch resolves a stable command for starting Lamha again.
package launch

import "os"

// Executable returns the persistent AppImage when Lamha is running from one.
// os.Executable points inside AppImage's temporary FUSE mount, which is not a
// usable command after logout, reboot, or an application restart.
func Executable() (string, error) {
	return executable(os.Getenv, os.Executable)
}

func executable(getenv func(string) string, resolve func() (string, error)) (string, error) {
	if appImage := getenv("APPIMAGE"); appImage != "" {
		return appImage, nil
	}
	return resolve()
}
