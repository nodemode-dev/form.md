//go:build windows

package agentlookup

import (
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var expandProc = syscall.NewLazyDLL("kernel32.dll").NewProc("ExpandEnvironmentStringsW")

func ownSession(cmd *exec.Cmd) {}

func registryValue(root syscall.Handle, key, name string) string {
	keyPtr, err := syscall.UTF16PtrFromString(key)
	if err != nil {
		return ""
	}
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return ""
	}
	var opened syscall.Handle
	if syscall.RegOpenKeyEx(root, keyPtr, 0, syscall.KEY_READ, &opened) != nil {
		return ""
	}
	defer syscall.RegCloseKey(opened)
	var kind, size uint32
	if syscall.RegQueryValueEx(opened, namePtr, nil, &kind, nil, &size) != nil || size < 2 {
		return ""
	}
	buf := make([]uint16, size/2+1)
	if syscall.RegQueryValueEx(opened, namePtr, nil, &kind, (*byte)(unsafe.Pointer(&buf[0])), &size) != nil {
		return ""
	}
	raw := syscall.UTF16ToString(buf)
	src, err := syscall.UTF16PtrFromString(raw)
	if err != nil {
		return raw
	}
	out := make([]uint16, 32768)
	n, _, _ := expandProc.Call(uintptr(unsafe.Pointer(src)), uintptr(unsafe.Pointer(&out[0])), uintptr(len(out)))
	if n == 0 || int(n) > len(out) {
		return raw
	}
	return syscall.UTF16ToString(out[:n])
}

func registryPath() []string {
	values := []string{
		registryValue(syscall.HKEY_CURRENT_USER, "Environment", "Path"),
		registryValue(syscall.HKEY_LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, "Path"),
	}
	out := []string{}
	for _, value := range values {
		for _, p := range strings.Split(value, ";") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
	}
	return unique(out)
}
