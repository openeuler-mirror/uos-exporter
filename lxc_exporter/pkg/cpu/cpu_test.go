package cpu

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseProcStat_Valid(t *testing.T) {
	input := "cpu  100 200 50 3000 20 10 5 0"
	expected := ProcStat{
		User:       100,
		System:     200,
		Nice:       50,
		Idle:       3000,
		Wait:       20,
		Irq:        10,
		Srq:        5,
		Zero:       0,
		prevUser:   100,
		prevSystem: 200,
		prevNice:   50,
		prevIdle:   3000,
		prevWait:   20,
		prevIrq:    10,
		prevSrq:    5,
		prevZero:   0,
	}

	stat, err := parseProcStat(input)
	assert.NoError(t, err)
	assert.Equal(t, expected, stat)
}

func TestParseProcStat_InvalidFormat(t *testing.T) {
	input := "invalid_data"
	_, err := parseProcStat(input)
	assert.Error(t, err)
}

func TestProcStatRefresh(t *testing.T) {
	initialStat := ProcStat{
		User:       100,
		System:     200,
		Nice:       50,
		Idle:       3000,
		Wait:       20,
		Irq:        10,
		Srq:        5,
		Zero:       0,
		prevUser:   100,
		prevSystem: 200,
		prevNice:   50,
		prevIdle:   3000,
		prevWait:   20,
		prevIrq:    10,
		prevSrq:    5,
		prevZero:   0,
	}

	//newData := "cpu  110 210 55 3010 25 12 7 0"
	//newStat, err := parseProcStat(newData)
	//assert.NoError(t, err)

	initialStat.Refresh()
	//
	//assert.Equal(t, 10.0, initialStat.User)
	//assert.Equal(t, 10.0, initialStat.System)
	//assert.Equal(t, 5.0, initialStat.Nice)
	//assert.Equal(t, 10.0, initialStat.Idle)
	//assert.Equal(t, 5.0, initialStat.Wait)
	//assert.Equal(t, 2.0, initialStat.Irq)
	//assert.Equal(t, 2.0, initialStat.Srq)
	//assert.Equal(t, 0.0, initialStat.Zero)
}

func TestFetchProcStat_FileNotFound(t *testing.T) {

	originalPath := procStatPath
	procStatPath = "/invalid/path"
	defer func() { procStatPath = originalPath }()

	_, err := fetchProcStat()
	assert.Error(t, err)
}
