//go:build windows && arm64

TEXT ·NtAllocateVirtualMemory(SB), $0-56
    MOVD hProcess+0(FP), X0
    MOVD baseAddr+8(FP), X1
    MOVD zeroBits+16(FP), X2
    MOVD regionSize+24(FP), X3
    MOVD allocType+32(FP), X4
    MOVD protect+40(FP), X5
    MOVW sysID+48(FP), W16
    SVC #0
    MOVW R0, ret+52(FP)
    RET

TEXT ·NtWriteVirtualMemory(SB), $0-64
    MOVD hProcess+0(FP), X0
    MOVD baseAddr+8(FP), X1
    MOVD buffer+16(FP), X2
    MOVD size+40(FP), X3
    MOVD $0, X4
    MOVW sysID+48(FP), W16
    SVC #0
    MOVW R0, ret+56(FP)
    RET

TEXT ·NtProtectVirtualMemory(SB), $0-48
    MOVD hProcess+0(FP), X0
    MOVD baseAddr+8(FP), X1
    MOVD size+16(FP), X2
    MOVD newProtect+24(FP), X3
    MOVD oldProtect+32(FP), X4
    MOVW sysID+40(FP), W16
    SVC #0
    MOVW R0, ret+44(FP)
    RET

TEXT ·NtGetContextThread(SB), $0-32
    MOVD hThread+0(FP), X0
    MOVD lpContext+8(FP), X1
    MOVW sysID+16(FP), W16
    SVC #0
    MOVW R0, ret+24(FP)
    RET

TEXT ·NtSetContextThread(SB), $0-32
    MOVD hThread+0(FP), X0
    MOVD lpContext+8(FP), X1
    MOVW sysID+16(FP), W16
    SVC #0
    MOVW R0, ret+24(FP)
    RET
