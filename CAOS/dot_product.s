    .intel_syntax noprefix

    .text
    .global dot_product

dot_product:
    pxor xmm0, xmm0

while:
    cmp rdi, 0
    jle end

    vmovups xmm1, [rsi]
    vmovups xmm2, [rdx]
    vmulps xmm1, xmm1, xmm2
    haddps xmm1, xmm1
    haddps xmm1, xmm1
    addss xmm0, xmm1

    add rsi, 16
    add rdx, 16
    sub rdi, 4
    jmp while

end:
    ret