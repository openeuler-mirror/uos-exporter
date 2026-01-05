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


// TODO: implement functions
