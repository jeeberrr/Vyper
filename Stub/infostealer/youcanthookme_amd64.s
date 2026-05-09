//go:build windows && amd64

TEXT ·NtAllocateVirtualMemory(SB), $0-56
    MOVQ hProcess+0(FP), R10
    MOVQ baseAddr+8(FP), DX
    MOVQ zeroBits+16(FP), R8
    MOVQ regionSize+24(FP), R9
    MOVQ allocType+32(FP), R11
    MOVQ R11, 32(SP)
    MOVQ protect+40(FP), R11
    MOVQ R11, 40(SP)
    MOVL sysID+48(FP), AX
    SYSCALL
    MOVL AX, ret+52(FP)
    RET

TEXT ·NtWriteVirtualMemory(SB), $0-64
    MOVQ hProcess+0(FP), R10
    MOVQ baseAddr+8(FP), DX
    MOVQ buffer+16(FP), R8
    MOVQ size+40(FP), R9
    MOVQ $0, R11
    MOVQ R11, 32(SP)
    MOVL sysID+48(FP), AX
    SYSCALL
    MOVL AX, ret+56(FP)
    RET

TEXT ·NtProtectVirtualMemory(SB), $0-48
    MOVQ hProcess+0(FP), R10
    MOVQ baseAddr+8(FP), DX
    MOVQ size+16(FP), R8
    MOVQ newProtect+24(FP), R9
    MOVQ oldProtect+32(FP), R11
    MOVQ R11, 32(SP)
    MOVL sysID+40(FP), AX
    SYSCALL
    MOVL AX, ret+44(FP)
    RET

TEXT ·NtGetContextThread(SB), $0-32
    MOVQ hThread+0(FP), R10
    MOVQ lpContext+8(FP), DX
    MOVL sysID+16(FP), AX
    SYSCALL
    MOVL AX, ret+24(FP)
    RET

TEXT ·NtSetContextThread(SB), $0-32
    MOVQ hThread+0(FP), R10
    MOVQ lpContext+8(FP), DX
    MOVL sysID+16(FP), AX
    SYSCALL
    MOVL AX, ret+24(FP)
    RET
