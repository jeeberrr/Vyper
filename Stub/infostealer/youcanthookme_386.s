//go:build windows && 386

TEXT ·NtAllocateVirtualMemory(SB), $0-32
    MOVL sysID+24(FP), AX
    LEAL hProcess+0(FP), EDX
    INT $0x2E
    MOVL AX, ret+28(FP)
    RET

TEXT ·NtWriteVirtualMemory(SB), $0-36
    MOVL sysID+32(FP), AX
    LEAL hProcess+0(FP), EDX
    INT $0x2E
    MOVL AX, ret+36(FP)
    RET

TEXT ·NtProtectVirtualMemory(SB), $0-28
    MOVL sysID+24(FP), AX
    LEAL hProcess+0(FP), EDX
    INT $0x2E
    MOVL AX, ret+28(FP)
    RET

TEXT ·NtGetContextThread(SB), $0-16
    MOVL sysID+8(FP), AX
    LEAL hThread+0(FP), EDX
    INT $0x2E
    MOVL AX, ret+12(FP)
    RET

TEXT ·NtSetContextThread(SB), $0-16
    MOVL sysID+8(FP), AX
    LEAL hThread+0(FP), EDX
    INT $0x2E
    MOVL AX, ret+12(FP)
    RET
