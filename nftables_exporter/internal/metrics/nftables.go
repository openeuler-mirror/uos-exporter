package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"nftables_exporter/internal/exporter"
	"nftables_exporter/internal/nftables"
)

func init() {
	exporter.Register(
		NewNftables("nft_summary_info",
			"nft summary info",
			[]string{"type"}))
}

type Nftables struct {
	*baseMetrics
}

func NewNftables(fqname, help string, labels []string) *Nftables {
	return &Nftables{NewMetrics(fqname, help, labels)}
}

func (c *Nftables) Collect(ch chan<- prometheus.Metric) {
	nft := nftables.New()
	err := nft.Update()
	if err != nil {
		logrus.Errorf("get nft summary failed: %v", err)
		return
	}
	c.baseMetrics.collect(ch,
		float64(nft.GetTableCount()),
		[]string{"table"})
	c.baseMetrics.collect(ch,
		float64(nft.GetChainCount()),
		[]string{"chain"})
	c.baseMetrics.collect(ch, float64(nft.GetRuleCount()), []string{"rule"})
}
// Part 2 commit for nftables_exporter/internal/metrics/nftables.go
