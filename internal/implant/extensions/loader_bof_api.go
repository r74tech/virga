//go:build !purego
// +build !purego

// Package extensions provides BOF (Beacon Object File) API implementation.
// This file contains unsafe pointer operations that are required for
// compatibility with BOF calling conventions. The unsafe operations are
// necessary to provide function pointers that can be called from loaded
// BOF code.
//
// WARNING: This code uses unsafe pointers in ways that violate Go's unsafe
// pointer rules. This is intentional and required for BOF compatibility.
// Do not use this code as an example for general Go programming.

//nolint:all // This entire file uses unsafe pointers for BOF API compatibility
//revive:disable // Unsafe pointer usage is intentional for BOF compatibility

package extensions

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"unsafe"
)

// BOFContext represents the execution context for a BOF
type BOFContext struct {
	extension   *BOFExtension
	callbacks   *ExtensionCallbacks
	output      *bytes.Buffer
	dataParsers map[uintptr]*BOFDataParser
	mu          sync.Mutex
}

// BOFDataParser helps parse BOF arguments
type BOFDataParser struct {
	data   []byte
	offset int
}

// CreateBOFAPI creates the BOF API function table
func CreateBOFAPI(ctx *BOFContext) map[string]uintptr {
	api := make(map[string]uintptr)

	// Output functions
	api["BeaconPrintf"] = createBeaconPrintf(ctx)
	api["BeaconOutput"] = createBeaconOutput(ctx)

	// Data parsing functions
	api["BeaconDataParse"] = createBeaconDataParse(ctx)
	api["BeaconDataInt"] = createBeaconDataInt(ctx)
	api["BeaconDataShort"] = createBeaconDataShort(ctx)
	api["BeaconDataLength"] = createBeaconDataLength(ctx)
	api["BeaconDataExtract"] = createBeaconDataExtract(ctx)

	// Format functions
	api["BeaconFormatAlloc"] = createBeaconFormatAlloc(ctx)
	api["BeaconFormatReset"] = createBeaconFormatReset(ctx)
	api["BeaconFormatFree"] = createBeaconFormatFree(ctx)
	api["BeaconFormatAppend"] = createBeaconFormatAppend(ctx)
	api["BeaconFormatPrintf"] = createBeaconFormatPrintf(ctx)
	api["BeaconFormatToString"] = createBeaconFormatToString(ctx)
	api["BeaconFormatInt"] = createBeaconFormatInt(ctx)

	// Token functions
	api["BeaconUseToken"] = createBeaconUseToken(ctx)
	api["BeaconRevertToken"] = createBeaconRevertToken(ctx)
	api["BeaconIsAdmin"] = createBeaconIsAdmin(ctx)

	// Process functions
	api["BeaconGetSpawnTo"] = createBeaconGetSpawnTo(ctx)
	api["BeaconInjectProcess"] = createBeaconInjectProcess(ctx)
	api["BeaconInjectTemporaryProcess"] = createBeaconInjectTemporaryProcess(ctx)
	api["BeaconCleanupProcess"] = createBeaconCleanupProcess(ctx)

	// Utility functions
	api["toWideChar"] = createToWideChar(ctx)

	return api
}

// BeaconPrintf implementation
func createBeaconPrintf(ctx *BOFContext) uintptr {
	// This is a simplified version - in reality, we'd need to create proper function stubs
	// that can be called from the BOF code
	fn := func(format uintptr, args ...uintptr) {
		// In a real implementation, this would parse the format string and arguments
		// from the BOF's memory space
		ctx.mu.Lock()
		defer ctx.mu.Unlock()

		// For now, just write a placeholder
		ctx.output.WriteString("[BeaconPrintf output]\n")
	}

	// Return function pointer
	// In a real implementation, this would return a pointer to assembly code
	// that implements the calling convention expected by the BOF
	return uintptr(unsafe.Pointer(&fn))
}

// BeaconOutput implementation
func createBeaconOutput(ctx *BOFContext) uintptr {
	fn := func(data uintptr, length int32) {
		ctx.mu.Lock()
		defer ctx.mu.Unlock()

		// Copy data from BOF memory
		output := make([]byte, length)
		// In a real implementation, we'd copy from the BOF's memory space
		// For now, just acknowledge the output
		ctx.output.Write(output)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconDataParse implementation
func createBeaconDataParse(ctx *BOFContext) uintptr {
	fn := func(data uintptr, length int32) uintptr {
		// Create a new data parser
		parser := &BOFDataParser{
			data:   make([]byte, length),
			offset: 0,
		}

		// Copy data from BOF memory
		// In a real implementation, we'd copy from the BOF's memory space

		// Store parser and return handle
		ctx.mu.Lock()
		handle := uintptr(unsafe.Pointer(parser)) //nolint:govet
		ctx.dataParsers[handle] = parser
		ctx.mu.Unlock()

		return handle
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconDataInt implementation
func createBeaconDataInt(ctx *BOFContext) uintptr {
	fn := func(parserHandle uintptr) int32 {
		ctx.mu.Lock()
		parser, ok := ctx.dataParsers[parserHandle]
		ctx.mu.Unlock()

		if !ok || parser.offset+4 > len(parser.data) {
			return 0
		}

		value := binary.LittleEndian.Uint32(parser.data[parser.offset:])
		parser.offset += 4
		return int32(value)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconDataShort implementation
func createBeaconDataShort(ctx *BOFContext) uintptr {
	fn := func(parserHandle uintptr) int16 {
		ctx.mu.Lock()
		parser, ok := ctx.dataParsers[parserHandle]
		ctx.mu.Unlock()

		if !ok || parser.offset+2 > len(parser.data) {
			return 0
		}

		value := binary.LittleEndian.Uint16(parser.data[parser.offset:])
		parser.offset += 2
		return int16(value)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconDataLength implementation
func createBeaconDataLength(ctx *BOFContext) uintptr {
	fn := func(parserHandle uintptr) int32 {
		ctx.mu.Lock()
		parser, ok := ctx.dataParsers[parserHandle]
		ctx.mu.Unlock()

		if !ok || parser.offset+4 > len(parser.data) {
			return 0
		}

		// Read length without advancing offset
		length := binary.LittleEndian.Uint32(parser.data[parser.offset:])
		return int32(length)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconDataExtract implementation
func createBeaconDataExtract(ctx *BOFContext) uintptr {
	fn := func(parserHandle uintptr, lengthPtr uintptr) uintptr {
		ctx.mu.Lock()
		parser, ok := ctx.dataParsers[parserHandle]
		ctx.mu.Unlock()

		if !ok || parser.offset+4 > len(parser.data) {
			return 0
		}

		// Read length
		length := binary.LittleEndian.Uint32(parser.data[parser.offset:])
		parser.offset += 4

		if parser.offset+int(length) > len(parser.data) {
			return 0
		}

		// Extract data
		data := parser.data[parser.offset : parser.offset+int(length)]
		parser.offset += int(length)

		// Write length if pointer provided
		if lengthPtr != 0 {
			// #nosec G103 -- Required for BOF API compatibility
			*(*int32)(unsafe.Pointer(lengthPtr)) = int32(length) //nolint:govet // Required for BOF API
		}

		// Return pointer to data
		return uintptr(unsafe.Pointer(&data[0])) //nolint:govet
	}

	return uintptr(unsafe.Pointer(&fn))
}

// Format buffer for BeaconFormat* functions
type FormatBuffer struct {
	data []byte
}

// BeaconFormatAlloc implementation
func createBeaconFormatAlloc(_ *BOFContext) uintptr {
	fn := func() uintptr {
		buffer := &FormatBuffer{
			data: make([]byte, 0, 1024),
		}
		return uintptr(unsafe.Pointer(buffer)) //nolint:govet
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconFormatReset implementation
func createBeaconFormatReset(_ *BOFContext) uintptr {
	fn := func(bufferPtr uintptr) {
		if bufferPtr == 0 {
			return
		}
		buffer := (*FormatBuffer)(unsafe.Pointer(bufferPtr)) //nolint:govet // Required for BOF API
		buffer.data = buffer.data[:0]
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconFormatFree implementation
func createBeaconFormatFree(_ *BOFContext) uintptr {
	fn := func(bufferPtr uintptr) {
		// In a real implementation, we'd free the memory
		// Go's GC will handle this for us
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconFormatAppend implementation
func createBeaconFormatAppend(_ *BOFContext) uintptr {
	fn := func(bufferPtr uintptr, data uintptr, length int32) {
		if bufferPtr == 0 {
			return
		}
		buffer := (*FormatBuffer)(unsafe.Pointer(bufferPtr)) //nolint:govet // Required for BOF API

		// Append data to buffer
		newData := make([]byte, length)
		// In a real implementation, copy from BOF memory
		buffer.data = append(buffer.data, newData...)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconFormatPrintf implementation
func createBeaconFormatPrintf(_ *BOFContext) uintptr {
	fn := func(bufferPtr uintptr, format uintptr, args ...uintptr) {
		if bufferPtr == 0 {
			return
		}
		buffer := (*FormatBuffer)(unsafe.Pointer(bufferPtr)) //nolint:govet // Required for BOF API

		// In a real implementation, parse format string and format data
		formatted := "[Formatted output]"
		buffer.data = append(buffer.data, []byte(formatted)...)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconFormatToString implementation
func createBeaconFormatToString(_ *BOFContext) uintptr {
	fn := func(bufferPtr uintptr, lengthPtr uintptr) uintptr {
		if bufferPtr == 0 {
			return 0
		}
		buffer := (*FormatBuffer)(unsafe.Pointer(bufferPtr)) //nolint:govet // Required for BOF API

		if lengthPtr != 0 {
			*(*int32)(unsafe.Pointer(lengthPtr)) = int32(len(buffer.data)) //nolint:govet // Required for BOF API
		}

		if len(buffer.data) == 0 {
			return 0
		}

		return uintptr(unsafe.Pointer(&buffer.data[0])) //nolint:govet // Required for BOF API
	}

	return uintptr(unsafe.Pointer(&fn))
}

// BeaconFormatInt implementation
func createBeaconFormatInt(_ *BOFContext) uintptr {
	fn := func(bufferPtr uintptr, value int32) {
		if bufferPtr == 0 {
			return
		}
		buffer := (*FormatBuffer)(unsafe.Pointer(bufferPtr)) //nolint:govet // Required for BOF API

		// Append int as 4 bytes
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], uint32(value))
		buffer.data = append(buffer.data, buf[:]...)
	}

	return uintptr(unsafe.Pointer(&fn))
}

// Token functions
func createBeaconUseToken(_ *BOFContext) uintptr {
	fn := func(token uintptr) bool {
		// In a real implementation, this would impersonate a token
		return true
	}

	return uintptr(unsafe.Pointer(&fn))
}

func createBeaconRevertToken(_ *BOFContext) uintptr {
	fn := func() {
		// In a real implementation, this would revert token impersonation
	}

	return uintptr(unsafe.Pointer(&fn))
}

func createBeaconIsAdmin(_ *BOFContext) uintptr {
	fn := func() bool {
		// In a real implementation, check if running with admin privileges
		return false
	}

	return uintptr(unsafe.Pointer(&fn))
}

// Process functions
func createBeaconGetSpawnTo(_ *BOFContext) uintptr {
	fn := func(x86 bool) uintptr {
		// Return path to process to spawn
		if x86 {
			return uintptr(unsafe.Pointer(&[]byte("C:\\Windows\\SysWOW64\\rundll32.exe\x00")[0])) //nolint:govet
		}
		return uintptr(unsafe.Pointer(&[]byte("C:\\Windows\\System32\\rundll32.exe\x00")[0])) //nolint:govet
	}

	return uintptr(unsafe.Pointer(&fn))
}

func createBeaconInjectProcess(ctx *BOFContext) uintptr {
	fn := func(pid int32, data uintptr, length int32) {
		// In a real implementation, inject into process
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		fmt.Fprintf(ctx.output, "[Would inject %d bytes into PID %d]\n", length, pid)
	}

	return uintptr(unsafe.Pointer(&fn))
}

func createBeaconInjectTemporaryProcess(ctx *BOFContext) uintptr {
	fn := func(data uintptr, length int32) {
		// In a real implementation, spawn process and inject
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		fmt.Fprintf(ctx.output, "[Would spawn process and inject %d bytes]\n", length)
	}

	return uintptr(unsafe.Pointer(&fn))
}

func createBeaconCleanupProcess(_ *BOFContext) uintptr {
	fn := func(process uintptr) {
		// In a real implementation, cleanup spawned process
	}

	return uintptr(unsafe.Pointer(&fn))
}

// Utility functions
func createToWideChar(_ *BOFContext) uintptr {
	fn := func(str uintptr) uintptr {
		// In a real implementation, convert to wide string
		return str
	}

	return uintptr(unsafe.Pointer(&fn))
}

// ExecuteBOF executes a loaded BOF
func ExecuteBOF(loaded *LoadedBOF, args []byte, ctx *BOFContext) error {
	if loaded.EntryPoint == 0 {
		return fmt.Errorf("no entry point found")
	}

	// In a real implementation, we would:
	// 1. Set up the execution environment
	// 2. Push arguments onto the stack or into registers
	// 3. Call the entry point
	// 4. Capture the output

	// For now, we'll simulate execution
	ctx.output.WriteString("[BOF execution simulated]\n")
	fmt.Fprintf(ctx.output, "Entry point: 0x%x\n", loaded.EntryPoint)
	fmt.Fprintf(ctx.output, "Arguments size: %d bytes\n", len(args))

	// In a real implementation on Windows, we would use something like:
	// - VirtualAlloc to allocate executable memory
	// - Copy BOF code and apply relocations
	// - Set up stack with arguments
	// - Call the entry point using assembly or CreateThread

	return nil
}
