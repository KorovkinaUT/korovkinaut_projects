  .text
  .global longest_inc_subseq

longest_inc_subseq:
  cmp x2, 0
  b.ls size_0

  mov x8, 0 // i
  mov x9, 1
  str x9, [x1]
first_for:
  add x8, x8, 1 // ++i
  cmp x2, x8
  b.ls after_for // size <= i

  ldr x10, [x0, x8, lsl 3]
  mov x9, 0 // j
  mov x11, 1
  str x11, [x1, x8, lsl 3]
inner_for:
  ldr x11, [x0, x9, lsl 3]
  cmp x10, x11
  b.le after_if // array[i] <= array[j]

  ldr x12, [x1, x9, lsl 3]
  add x12, x12, 1
  ldr x13, [x1, x8, lsl 3]
  cmp x12, x13
  b.ls after_if // dp[j] <= dp[i]

  str x12, [x1, x8, lsl 3]

after_if:
  add x9, x9, 1 // ++j
  cmp x8, x9
  b.ls first_for // i <= j
  b inner_for

after_for:
  mov x8, 0
  mov x0, 1
second_for:
  add x8, x8, 1 // ++i
  cmp x2, x8
  b.ls end // size <= i

  ldr x9, [x1, x8, lsl 3]
  cmp x9, x0
  b.ls second_for // current_dp <= current_max

  mov x0, x9
  b second_for

size_0:
  mov x0, 0
  // b end

end:
  ret
