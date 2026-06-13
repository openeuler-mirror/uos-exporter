package lxc

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	memoryStatFile = "memory.stat"
)

type MemoryStat struct {
	Anon                   float64
	File                   float64
	Kernel                 float64
	KernelStack            float64
	Pagetables             float64
	SecPagetables          float64
	Percpu                 float64
	Sock                   float64
	Vmalloc                float64
	Shmem                  float64
	Zswap                  float64
	Zswapped               float64
	FileMapped             float64
	FileDirty              float64
	FileWriteback          float64
	Swapcached             float64
	AnonThp                float64
	FileThp                float64
	ShmemThp               float64
	InactiveAnon           float64
	ActiveAnon             float64
	InactiveFile           float64
	ActiveFile             float64
	Unevictable            float64
	SlabReclaimable        float64
	SlabUnreclaimable      float64
	Slab                   float64
	WorkingsetRefaultAnon  float64
	WorkingsetRefaultFile  float64
	WorkingsetActivateAnon float64
	WorkingsetActivateFile float64
	WorkingsetRestoreAnon  float64
	WorkingsetRestoreFile  float64
	WorkingsetNodereclaim  float64
	Pgscan                 float64
	Pgsteal                float64
	PgscanKswapd           float64
	PgscanDirect           float64
	PgscanKhugepaged       float64
	PgstealKswapd          float64
	PgstealDirect          float64
	PgstealKhugepaged      float64
	Pgfault                float64
	Pgmajfault             float64
	Pgrefill               float64
	Pgactivate             float64
	Pgdeactivate           float64
	Pglazyfree             float64
	Pglazyfreed            float64
	Zswpin                 float64
	Zswpout                float64
	ThpFaultAlloc          float64
	ThpCollapseAlloc       float64
}

func (l *Lxc) GetMemoryStat(containerName string) (MemoryStat, error) {
	if !l.containerExists(containerName) {
		return MemoryStat{}, ErrorContainerNotFound
	}
	statContent, err := l.readMemoryStatFile(containerName)
	if err != nil {
		return MemoryStat{}, err
	}
	return parseMemoryStat(statContent)
}
func (l *Lxc) GetMemoryStatAll() ([]MemoryStat, error) {
	return nil, nil
}

func (l *Lxc) readMemoryStatFile(containerName string) ([]byte, error) {
	cgroupStatPath, err := l.getMemoryStatFilePath(containerName)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(cgroupStatPath)
	if !strings.HasPrefix(cleanPath, cgroupPath) {
		return nil, fmt.Errorf("config file must be located within %s", cgroupPath)
	}
	return os.ReadFile(cleanPath)
}

func (l *Lxc) getMemoryStatFilePath(containerName string) (string, error) {
	memStatPath := path.Join(cgroupPath,
		//lxcPrefix,
		l.getContainerPathName(containerName),
		memoryStatFile)
	_, err := os.Stat(memStatPath)
	if err != nil {
		return "", err
	}
	return memStatPath, nil
}

func parseMemoryStat(content []byte) (MemoryStat, error) {
	var (
		Anon                   float64
		File                   float64
		Kernel                 float64
		KernelStack            float64
		Pagetables             float64
		SecPagetables          float64
		Percpu                 float64
		Sock                   float64
		Vmalloc                float64
		Shmem                  float64
		Zswap                  float64
		Zswapped               float64
		FileMapped             float64
		FileDirty              float64
		FileWriteback          float64
		Swapcached             float64
		AnonThp                float64
		FileThp                float64
		ShmemThp               float64
		InactiveAnon           float64
		ActiveAnon             float64
		InactiveFile           float64
		ActiveFile             float64
		Unevictable            float64
		SlabReclaimable        float64
		SlabUnreclaimable      float64
		Slab                   float64
		WorkingsetRefaultAnon  float64
		WorkingsetRefaultFile  float64
		WorkingsetActivateAnon float64
		WorkingsetActivateFile float64
		WorkingsetRestoreAnon  float64
		WorkingsetRestoreFile  float64
		WorkingsetNodereclaim  float64
		Pgscan                 float64
		Pgsteal                float64
		PgscanKswapd           float64
		PgscanDirect           float64
		PgscanKhugepaged       float64
		PgstealKswapd          float64
		PgstealDirect          float64
		PgstealKhugepaged      float64
		Pgfault                float64
		Pgmajfault             float64
		Pgrefill               float64
		Pgactivate             float64
		Pgdeactivate           float64
		Pglazyfree             float64
		Pglazyfreed            float64
		Zswpin                 float64
		Zswpout                float64
		ThpFaultAlloc          float64
		ThpCollapseAlloc       float64
	)
	lines := strings.Split(
		strings.TrimSpace(
			string(content)), "\n")
	if len(lines) < 53 {
		return MemoryStat{},
			errors.New("invalid memory.stat format")
	}
	Anon, err := parseMemoryStatLine(lines[0])
	if err != nil {
		return MemoryStat{}, err
	}
	File, err = parseMemoryStatLine(lines[1])
	if err != nil {
		return MemoryStat{}, err
	}
	Kernel, err = parseMemoryStatLine(lines[2])
	if err != nil {
		return MemoryStat{}, err
	}
	KernelStack, err = parseMemoryStatLine(lines[3])
	if err != nil {
		return MemoryStat{}, err
	}
	Pagetables, err = parseMemoryStatLine(lines[4])
	if err != nil {
		return MemoryStat{}, err
	}
	SecPagetables, err = parseMemoryStatLine(lines[5])
	if err != nil {
		return MemoryStat{}, err
	}
	Percpu, err = parseMemoryStatLine(lines[6])
	if err != nil {
		return MemoryStat{}, err
	}
	Sock, err = parseMemoryStatLine(lines[7])
	if err != nil {
		return MemoryStat{}, err
	}
	Vmalloc, err = parseMemoryStatLine(lines[8])
	if err != nil {
		return MemoryStat{}, err
	}
	Shmem, err = parseMemoryStatLine(lines[9])
	if err != nil {
		return MemoryStat{}, err
	}
	Zswap, err = parseMemoryStatLine(lines[10])
	if err != nil {
		return MemoryStat{}, err
	}
	Zswapped, err = parseMemoryStatLine(lines[11])
	if err != nil {
		return MemoryStat{}, err
	}
	FileMapped, err = parseMemoryStatLine(lines[12])
	if err != nil {
		return MemoryStat{}, err
	}
	FileDirty, err = parseMemoryStatLine(lines[13])
	if err != nil {
		return MemoryStat{}, err
	}
	FileWriteback, err = parseMemoryStatLine(lines[14])
	if err != nil {
		return MemoryStat{}, err
	}
	Swapcached, err = parseMemoryStatLine(lines[15])
	if err != nil {
		return MemoryStat{}, err
	}
	AnonThp, err = parseMemoryStatLine(lines[16])
	if err != nil {
		return MemoryStat{}, err
	}
	FileThp, err = parseMemoryStatLine(lines[17])
	if err != nil {
		return MemoryStat{}, err
	}
	ShmemThp, err = parseMemoryStatLine(lines[18])
	if err != nil {
		return MemoryStat{}, err
	}
	InactiveAnon, err = parseMemoryStatLine(lines[19])
	if err != nil {
		return MemoryStat{}, err
	}
	ActiveAnon, err = parseMemoryStatLine(lines[20])
	if err != nil {
		return MemoryStat{}, err
	}
	InactiveFile, err = parseMemoryStatLine(lines[21])
	if err != nil {
		return MemoryStat{}, err
	}
	ActiveFile, err = parseMemoryStatLine(lines[22])
	if err != nil {
		return MemoryStat{}, err
	}
	Unevictable, err = parseMemoryStatLine(lines[23])
	if err != nil {
		return MemoryStat{}, err
	}
	SlabReclaimable, err = parseMemoryStatLine(lines[24])
	if err != nil {
		return MemoryStat{}, err
	}
	SlabUnreclaimable, err = parseMemoryStatLine(lines[25])
	if err != nil {
		return MemoryStat{}, err
	}
	Slab, err = parseMemoryStatLine(lines[26])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetRefaultAnon, err = parseMemoryStatLine(lines[27])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetRefaultFile, err = parseMemoryStatLine(lines[28])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetActivateAnon, err = parseMemoryStatLine(lines[29])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetActivateFile, err = parseMemoryStatLine(lines[30])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetRestoreAnon, err = parseMemoryStatLine(lines[31])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetRestoreFile, err = parseMemoryStatLine(lines[32])
	if err != nil {
		return MemoryStat{}, err
	}
	WorkingsetNodereclaim, err = parseMemoryStatLine(lines[33])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgscan, err = parseMemoryStatLine(lines[34])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgsteal, err = parseMemoryStatLine(lines[35])
	if err != nil {
		return MemoryStat{}, err
	}
	PgscanKswapd, err = parseMemoryStatLine(lines[36])
	if err != nil {
		return MemoryStat{}, err
	}
	PgscanDirect, err = parseMemoryStatLine(lines[37])
	if err != nil {
		return MemoryStat{}, err
	}
	PgscanKhugepaged, err = parseMemoryStatLine(lines[38])
	if err != nil {
		return MemoryStat{}, err
	}
	PgstealKswapd, err = parseMemoryStatLine(lines[39])
	if err != nil {
		return MemoryStat{}, err
	}
	PgstealDirect, err = parseMemoryStatLine(lines[40])
	if err != nil {
		return MemoryStat{}, err
	}
	PgstealKhugepaged, err = parseMemoryStatLine(lines[41])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgfault, err = parseMemoryStatLine(lines[42])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgmajfault, err = parseMemoryStatLine(lines[43])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgrefill, err = parseMemoryStatLine(lines[44])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgactivate, err = parseMemoryStatLine(lines[45])
	if err != nil {
		return MemoryStat{}, err
	}
	Pgdeactivate, err = parseMemoryStatLine(lines[46])
	if err != nil {
		return MemoryStat{}, err
	}
	Pglazyfree, err = parseMemoryStatLine(lines[47])
	if err != nil {
		return MemoryStat{}, err
	}
	Pglazyfreed, err = parseMemoryStatLine(lines[48])
	if err != nil {
		return MemoryStat{}, err
	}
	Zswpin, err = parseMemoryStatLine(lines[49])
	if err != nil {
		return MemoryStat{}, err
	}
	Zswpout, err = parseMemoryStatLine(lines[50])
	if err != nil {
		return MemoryStat{}, err
	}
	ThpFaultAlloc, err = parseMemoryStatLine(lines[51])
	if err != nil {
		return MemoryStat{}, err
	}
	ThpCollapseAlloc, err = parseMemoryStatLine(lines[52])
	if err != nil {
		return MemoryStat{}, err
	}
	return MemoryStat{
		Anon:                   Anon,
		File:                   File,
		Kernel:                 Kernel,
		KernelStack:            KernelStack,
		Pagetables:             Pagetables,
		SecPagetables:          SecPagetables,
		Percpu:                 Percpu,
		Sock:                   Sock,
		Vmalloc:                Vmalloc,
		Shmem:                  Shmem,
		Zswap:                  Zswap,
		Zswapped:               Zswapped,
		FileMapped:             FileMapped,
		FileDirty:              FileDirty,
		FileWriteback:          FileWriteback,
		Swapcached:             Swapcached,
		AnonThp:                AnonThp,
		FileThp:                FileThp,
		ShmemThp:               ShmemThp,
		InactiveAnon:           InactiveAnon,
		ActiveAnon:             ActiveAnon,
		InactiveFile:           InactiveFile,
		ActiveFile:             ActiveFile,
		Unevictable:            Unevictable,
		SlabReclaimable:        SlabReclaimable,
		SlabUnreclaimable:      SlabUnreclaimable,
		Slab:                   Slab,
		WorkingsetRefaultAnon:  WorkingsetRefaultAnon,
		WorkingsetRefaultFile:  WorkingsetRefaultFile,
		WorkingsetActivateAnon: WorkingsetActivateAnon,
		WorkingsetActivateFile: WorkingsetActivateFile,
		WorkingsetRestoreAnon:  WorkingsetRestoreAnon,
		WorkingsetRestoreFile:  WorkingsetRestoreFile,
		WorkingsetNodereclaim:  WorkingsetNodereclaim,
		Pgscan:                 Pgscan,
		Pgsteal:                Pgsteal,
		PgscanKswapd:           PgscanKswapd,
		PgscanDirect:           PgscanDirect,
		PgscanKhugepaged:       PgscanKhugepaged,
		PgstealKswapd:          PgstealKswapd,
		PgstealDirect:          PgstealDirect,
		PgstealKhugepaged:      PgstealKhugepaged,
		Pgfault:                Pgfault,
		Pgmajfault:             Pgmajfault,
		Pgrefill:               Pgrefill,
		Pgactivate:             Pgactivate,
		Pgdeactivate:           Pgdeactivate,
		Pglazyfree:             Pglazyfree,
		Pglazyfreed:            Pglazyfreed,
		Zswpin:                 Zswpin,
		Zswpout:                Zswpout,
		ThpFaultAlloc:          ThpFaultAlloc,
		ThpCollapseAlloc:       ThpCollapseAlloc,
	}, nil

}

func parseMemoryStatLine(line string) (float64, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return 0,
			errors.New("invalid cpu.stat line format")
	}
	return parseFloat64(fields[1])
}
