package lxc

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseMemoryStat(t *testing.T) {
	validContent := []byte(
		`anon 65536
file 4096
kernel 110592
kernel_stack 0
pagetables 24576
sec_pagetables 0
percpu 208
sock 0
vmalloc 4096
shmem 4096
zswap 0
zswapped 0
file_mapped 0
file_dirty 0
file_writeback 0
swapcached 0
anon_thp 0
file_thp 0
shmem_thp 0
inactive_anon 0
active_anon 69632
inactive_file 0
active_file 0
unevictable 0
slab_reclaimable 27520
slab_unreclaimable 37296
slab 64816
workingset_refault_anon 0
workingset_refault_file 0
workingset_activate_anon 0
workingset_activate_file 0
workingset_restore_anon 0
workingset_restore_file 0
workingset_nodereclaim 0
pgscan 1
pgsteal 1
pgscan_kswapd 1
pgscan_direct 0
pgscan_khugepaged 0
pgsteal_kswapd 1
pgsteal_direct 0
pgsteal_khugepaged 0
pgfault 287
pgmajfault 0
pgrefill 0
pgactivate 0
pgdeactivate 0
pglazyfree 0
pglazyfreed 0
zswpin 0
zswpout 0
thp_fault_alloc 0
thp_collapse_alloc 0`)

	// 正确解析
	memStat, err := parseMemoryStat(validContent)
	assert.NoError(t, err)
	assert.Equal(t, float64(65536), memStat.Anon)
	assert.Equal(t, float64(4096), memStat.File)
	assert.Equal(t, float64(110592), memStat.Kernel)
	assert.Equal(t, float64(0), memStat.KernelStack)
	assert.Equal(t, float64(24576), memStat.Pagetables)
	assert.Equal(t, float64(208), memStat.Percpu)
	assert.Equal(t, float64(0), memStat.Sock)
	assert.Equal(t, float64(4096), memStat.Vmalloc)
	assert.Equal(t, float64(4096), memStat.Shmem)
	assert.Equal(t, float64(0), memStat.Zswap)
	assert.Equal(t, float64(0), memStat.Zswapped)
	assert.Equal(t, float64(0), memStat.FileMapped)
	assert.Equal(t, float64(0), memStat.FileDirty)
	assert.Equal(t, float64(0), memStat.FileWriteback)
	assert.Equal(t, float64(0), memStat.Swapcached)
	assert.Equal(t, float64(0), memStat.AnonThp)
	assert.Equal(t, float64(0), memStat.FileThp)
	assert.Equal(t, float64(0), memStat.ShmemThp)
	assert.Equal(t, float64(0), memStat.InactiveAnon)
	assert.Equal(t, float64(69632), memStat.ActiveAnon)
	assert.Equal(t, float64(0), memStat.InactiveFile)
	assert.Equal(t, float64(0), memStat.ActiveFile)
	assert.Equal(t, float64(0), memStat.Unevictable)
	assert.Equal(t, float64(27520), memStat.SlabReclaimable)
	assert.Equal(t, float64(37296), memStat.SlabUnreclaimable)
	assert.Equal(t, float64(64816), memStat.Slab)
	assert.Equal(t, float64(0), memStat.WorkingsetRefaultAnon)
	assert.Equal(t, float64(0), memStat.WorkingsetRefaultFile)
	assert.Equal(t, float64(0), memStat.WorkingsetActivateAnon)
	assert.Equal(t, float64(0), memStat.WorkingsetActivateFile)
	assert.Equal(t, float64(0), memStat.WorkingsetRestoreAnon)
	assert.Equal(t, float64(0), memStat.WorkingsetRestoreFile)
	assert.Equal(t, float64(0), memStat.WorkingsetNodereclaim)
	assert.Equal(t, float64(1), memStat.Pgscan)
	assert.Equal(t, float64(1), memStat.Pgsteal)
	assert.Equal(t, float64(0), memStat.PgscanDirect)
	assert.Equal(t, float64(1), memStat.PgscanKswapd)
	assert.Equal(t, float64(0), memStat.PgscanKhugepaged)
	assert.Equal(t, float64(0), memStat.PgstealDirect)
	assert.Equal(t, float64(0), memStat.PgstealKhugepaged)
	assert.Equal(t, float64(287), memStat.Pgfault)
	assert.Equal(t, float64(0), memStat.Pgmajfault)
	assert.Equal(t, float64(0), memStat.Pgrefill)
	assert.Equal(t, float64(0), memStat.Pgactivate)
	assert.Equal(t, float64(0), memStat.Pgdeactivate)
	assert.Equal(t, float64(0), memStat.Pglazyfree)
	assert.Equal(t, float64(0), memStat.Pglazyfreed)
	assert.Equal(t, float64(0), memStat.Zswpin)
	assert.Equal(t, float64(0), memStat.Zswpout)
	assert.Equal(t, float64(0), memStat.ThpFaultAlloc)
	assert.Equal(t, float64(0), memStat.ThpCollapseAlloc)
	assert.Equal(t, float64(0), memStat.SecPagetables)
	assert.Equal(t, float64(0), memStat.Pglazyfree)
	assert.Equal(t, float64(0), memStat.Pglazyfreed)
	assert.Equal(t, float64(0), memStat.Zswap)
	assert.Equal(t, float64(0), memStat.Zswapped)
	assert.Equal(t, float64(0), memStat.Zswpin)
	assert.Equal(t, float64(0), memStat.Zswpout)
	assert.Equal(t, float64(0), memStat.ThpFaultAlloc)
	assert.Equal(t, float64(0), memStat.ThpCollapseAlloc)
	// 测试格式错误（缺少字段）
	invalidContent := []byte(`zswpin 0
zswpout 0`)
	_, err = parseMemoryStat(invalidContent)
	assert.Error(t, err)
}

// 测试 `parseCPUStatLine`
func TestParseMemoryStatLine(t *testing.T) {
	// 测试正常情况
	value, err := parseMemoryStatLine("usage_usec 1234567")
	assert.NoError(t, err)
	assert.Equal(t, float64(1234567), value)

	// 测试格式错误
	_, err = parseMemoryStatLine("usage_usec")
	assert.Error(t, err)

	// 测试无效数值
	_, err = parseMemoryStatLine("usage_usec abc")
	assert.Error(t, err)
}
