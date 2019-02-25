// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cpu implements processor feature detection for
// various CPU architectures.
package cpu

// CacheLinePad is used to pad structs to avoid false sharing.
type CacheLinePad struct{ _ [cacheLineSize]byte }

// X86 contains the supported CPU features of the
// current X86/AMD64 platform. If the current platform
// is not X86/AMD64 then all feature flags are false.
//
// X86 is padded to avoid false sharing. Further the HasAVX
// and HasAVX2 are only set if the OS supports XMM and YMM
// registers in addition to the CPUID feature bit being set.
var X86 struct {
	_            CacheLinePad
	HasAES       bool // AES hardware implementation (AES NI)
	HasADX       bool // Multi-precision add-carry instruction extensions
	HasAVX       bool // Advanced vector extension
	HasAVX2      bool // Advanced vector extension 2
	HasBMI1      bool // Bit manipulation instruction set 1
	HasBMI2      bool // Bit manipulation instruction set 2
	HasERMS      bool // Enhanced REP for MOVSB and STOSB
	HasFMA       bool // Fused-multiply-add instructions
	HasOSXSAVE   bool // OS supports XSAVE/XRESTOR for saving/restoring XMM registers.
	HasPCLMULQDQ bool // PCLMULQDQ instruction - most often used for AES-GCM
	HasPOPCNT    bool // Hamming weight instruction POPCNT.
	HasRDRAND    bool // RDRAND instruction (on-chip random number generator)
	HasRDSEED    bool // RDSEED instruction (on-chip random number generator)
	HasSSE2      bool // Streaming SIMD extension 2 (always available on amd64)
	HasSSE3      bool // Streaming SIMD extension 3
	HasSSSE3     bool // Supplemental streaming SIMD extension 3
	HasSSE41     bool // Streaming SIMD extension 4 and 4.1
	HasSSE42     bool // Streaming SIMD extension 4 and 4.2
	_            CacheLinePad
}

// ARM64 contains the supported CPU features of the
// current ARMv8(aarch64) platform. If the current platform
// is not arm64 then all feature flags are false.
var ARM64 struct {
	_           CacheLinePad
	HasFP       bool // Floating-point instruction set (always available)
	HasASIMD    bool // Advanced SIMD (always available)
	HasEVTSTRM  bool // Event stream support
	HasAES      bool // AES hardware implementation
	HasPMULL    bool // Polynomial multiplication instruction set
	HasSHA1     bool // SHA1 hardware implementation
	HasSHA2     bool // SHA2 hardware implementation
	HasCRC32    bool // CRC32 hardware implementation
	HasATOMICS  bool // Atomic memory operation instruction set
	HasFPHP     bool // Half precision floating-point instruction set
	HasASIMDHP  bool // Advanced SIMD half precision instruction set
	HasCPUID    bool // CPUID identification scheme registers
	HasASIMDRDM bool // Rounding double multiply add/subtract instruction set
	HasJSCVT    bool // Javascript conversion from floating-point to integer
	HasFCMA     bool // Floating-point multiplication and addition of complex numbers
	HasLRCPC    bool // Release Consistent processor consistent support
	HasDCPOP    bool // Persistent memory support
	HasSHA3     bool // SHA3 hardware implementation
	HasSM3      bool // SM3 hardware implementation
	HasSM4      bool // SM4 hardware implementation
	HasASIMDDP  bool // Advanced SIMD double precision instruction set
	HasSHA512   bool // SHA512 hardware implementation
	HasSVE      bool // Scalable Vector Extensions
	HasASIMDFHM bool // Advanced SIMD multiplication FP16 to FP32
	_           CacheLinePad
}

// PPC64 contains the supported CPU features of the current ppc64/ppc64le platforms.
// If the current platform is not ppc64/ppc64le then all feature flags are false.
//
// For ppc64/ppc64le, it is safe to check only for ISA level starting on ISA v3.00,
// since there are no optional categories. There are some exceptions that also
// require kernel support to work (DARN, SCV), so there are feature bits for
// those as well. The minimum processor requirement is POWER8 (ISA 2.07).
// The struct is padded to avoid false sharing.
var PPC64 struct {
	_        CacheLinePad
	HasDARN  bool // Hardware random number generator (requires kernel enablement)
	HasSCV   bool // Syscall vectored (requires kernel enablement)
	IsPOWER8 bool // ISA v2.07 (POWER8)
	IsPOWER9 bool // ISA v3.00 (POWER9)
	_        CacheLinePad
}
