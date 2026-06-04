//go:build windows && cgo && (ORT || ALL)

package main

/*
#cgo LDFLAGS: -L${SRCDIR}/resources/lib/windows -lntdll
typedef void (*medmemo_ntdll_symbol)(void);
void NtReadFile(void);
void RtlNtStatusToDosError(void);
void NtCreateFile(void);
void NtWriteFile(void);
void NtOpenFile(void);
void NtCreateNamedPipeFile(void);
medmemo_ntdll_symbol medmemo_ntdll_imports[] = {
	NtReadFile,
	RtlNtStatusToDosError,
	NtCreateFile,
	NtWriteFile,
	NtOpenFile,
	NtCreateNamedPipeFile,
};
*/
import "C"
