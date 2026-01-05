package corosync

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultParser_Parse_Success(t *testing.T) {
	parser := NewParser()

	cfgToolOutput := []byte(`Printing link status.
Local node ID 1084780051
Link ID 0
		addr	= 10.0.0.1
		status	= OK
Link ID 1
        addr      = 172.16.0.1
        status    = OK`)

	quorumToolOutput := []byte(`Quorum information
------------------
Date:             Sun Sep 29 16:10:37 2019
Quorum provider:  corosync_votequorum
Nodes:            2
Node ID:          1084780051
Ring ID:          1084780051.44
Quorate:          Yes

Votequorum information
----------------------
Expected votes:   232
Highest expected: 22
Total votes:      21
Quorum:           421
Flags:            2Node Quorate WaitForAll

Membership information
----------------------
	Nodeid      Votes Qdevice Name
1084780051          1      NR dma-dog-hana01 (local)
1084780052          1      A,V,NMW dma-dog-hana02`)

	status, err := parser.Parse(cfgToolOutput, quorumToolOutput)
	assert.NoError(t, err)
	assert.Equal(t, "1084780051", status.NodeId)
	assert.Equal(t, "1084780051.44", status.RingId)
	assert.True(t, status.Quorate)
	assert.EqualValues(t, QuorumVotes{232, 22, 21, 421}, status.QuorumVotes)

	assert.Len(t, status.Rings, 2)
	assert.Equal(t, "0", status.Rings[0].Number)
	assert.Equal(t, "10.0.0.1", status.Rings[0].Address)
	assert.False(t, status.Rings[0].Faulty)

	assert.Equal(t, "1", status.Rings[1].Number)
	assert.Equal(t, "172.16.0.1", status.Rings[1].Address)
	assert.False(t, status.Rings[1].Faulty)

	assert.Len(t, status.Members, 2)
	m := status.Members[0]
	assert.Exactly(t, "1084780051", m.Id)
	assert.Exactly(t, "dma-dog-hana01", m.Name)
	assert.Exactly(t, "NR", m.Qdevice)
	assert.True(t, m.Local)
	assert.EqualValues(t, 1, m.Votes)

	m = status.Members[1]
	assert.Exactly(t, "1084780052", m.Id)
	assert.Exactly(t, "dma-dog-hana02", m.Name)
	assert.Exactly(t, "A,V,NMW", m.Qdevice)
	assert.False(t, m.Local)
	assert.EqualValues(t, 1, m.Votes)
}

func TestDefaultParser_Parse_RingIdVariants(t *testing.T) {
	parser := NewParser()

	cases := []struct {
		desc       string
		ringIdLine string
		expected   string
	}{
		{"dot format", "Ring ID:          12345.67", "12345.67"},
		{"slash format", "Ring ID:          12345/67", "12345/67"},
		{"number only", "Ring ID:          100", "100"},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			input := []byte("Quorum information\n" + c.ringIdLine + "\n")
			id, err := parser.(*defaultParser).parseRingId(input)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, id)
		})
	}
}

func TestDefaultParser_Parse_FaultyRings_OKandFAULTY_MixedFormats(t *testing.T) {
	parser := NewParser()

	cfgToolOutput := []byte(`
Printing ring status.
	Local node ID 16777226
	RING ID 0
			id      = 10.0.0.1
			status  = Marking ringid 0 interface 10.0.0.1 FAULTY
	RING ID 1
			id      = 172.16.0.1
			status  = ring 1 active with no faults`)

	rings := parser.(*defaultParser).parseRings(cfgToolOutput)

	assert.Len(t, rings, 2)
	assert.True(t, rings[0].Faulty, "ring-0 should be faulty")
	assert.False(t, rings[1].Faulty, "ring-1 should not be faulty")
}

func TestDefaultParser_ParseNodeId_ErrorOnMissingPattern(t *testing.T) {
	parser := NewParser()
	out := []byte("no node id line here")
	id, err := parser.(*defaultParser).parseNodeId(out)
	assert.ErrorContains(t, err, "node ID pattern not found")
	assert.Empty(t, id)
}

func TestDefaultParser_ParseRingId_ErrorOnMissingPattern(t *testing.T) {
	parser := NewParser()
	out := []byte("no ring id line here")
	id, err := parser.(*defaultParser).parseRingId(out)
	assert.ErrorContains(t, err, "ring ID pattern not found")
	assert.Empty(t, id)
}

func TestDefaultParser_ParseQuorate_ErrorOnMissingPatternAndFalseValue(t *testing.T) {
	parser := NewParser()
	out := []byte("no quorate line here")
	ok, err := parser.(*defaultParser).parseQuorate(out)
	assert.ErrorContains(t, err, "quorate status pattern not found")
	out = []byte("Quorate: No")
	ok, err = parser.(*defaultParser).parseQuorate(out)
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestDefaultParser_ParseQuorumVotes_Errors(t *testing.T) {
	parser := NewParser()
	out := []byte("no vote lines here")
	_, err := parser.(*defaultParser).parseQuoromVotes(out)
	assert.ErrorContains(t, err, "quorum votes pattern not found")

	for i, data := range [][]byte{
		[]byte(`Expected votes:   x Highest expected: y Total votes: z Quorum: t`),
		[]byte(`Expected votes:   -100 Highest expected: -100 Total votes: -100 Quorum: -100`),
		[]byte(`
Expected votes:   ` + strconv.FormatUint(^uint64(0), 10) + `
Highest expected: ` + strconv.FormatUint(^uint64(0), 10) + `
Total votes:      ` + strconv.FormatUint(^uint64(0), 10) + `
Quorum:           ` + strconv.FormatUint(^uint64(0), 10) + "00" + `
`),
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			_, err := parser.(*defaultParser).parseQuoromVotes(data)
			assert.Error(t, err)
		})
	}
}

func TestDefaultParser_ParseMembersSection_ErrorsAndVariants(t *testing.T) {
	parser := NewParser()
	out := []byte("no membership section here")
	_, err := parser.(*defaultParser).parseMembers(out)
	assert.ErrorContains(t, err, "membership section not found")

	// 测试votes字段非法（非数字）
	out = []byte(`Membership information
----------------------
	Nodeid      Votes Qdevice Name
aabbccdd       xyz   NR host01`)
	_, err = parser.(*defaultParser).parseMembers(out)
	assert.ErrorContains(t, err, "member line parsing failed")

	// 测试正常IPv6和本地节点标记(local)的解析：
	out = []byte(`Membership information
----------------------
	Nodeid      Votes Qdevice Name
abcd         3     NR host01::fe80
abcd2        4     NR host02::fe80 (local)`)
	members, err := parser.(*defaultParser).parseMembers(out)
	assert.NoError(t, err)
	assert.Len(t, members, 2)
	assert.True(t, members[1].Local)
}
