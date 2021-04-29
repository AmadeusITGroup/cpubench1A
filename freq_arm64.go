package main

// ASMLoopCycles is the number of cycles of 1 iteration of the CountASM loop
const ASMLoopCycles = 1025.0

// NFREQ is the number of iterations. Adjust if it too slow/too fast.
const NFREQ = 1 << 34

func CountASM(n int64) int
