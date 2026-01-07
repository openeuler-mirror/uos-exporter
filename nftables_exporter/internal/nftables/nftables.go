package nftables

import (
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"os/exec"
)

func getNFTablesDetails() (gjson.Result, error) {
	logrus.Debug("Get NFTablesDetails info")
	cmd := exec.Command("nft", "-j", "list", "ruleset")
	output, err := cmd.Output()
	if err != nil {
		logrus.Error("Failed to get NFTablesDetails info: ", err)
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(output), nil
}

type NFT struct {
	details     gjson.Result
	error       error
	tablesCount int
	chainCount  int
	ruleCount   int
}

func New() *NFT {
	return &NFT{
		error:       nil,
		tablesCount: 0,
		chainCount:  0,
		ruleCount:   0,
	}
}

func (n *NFT) GetTableCount() int {
	return n.tablesCount
}

func (n *NFT) GetChainCount() int {
	return n.chainCount
}

func (n *NFT) GetRuleCount() int {
	return n.ruleCount
}

func (n *NFT) Update() error {
	datails, err := getNFTablesDetails()
	if err != nil {
		n.error = err
		return err
	}
	n.details = datails.Get("nftables")
	n.tablesCount = len(n.details.
		Get("#.table").
		Array())
	n.chainCount = len(n.details.
		Get("#.chain").
		Array())
	n.ruleCount = len(n.details.
		Get("#.rule").
		Array())
	return nil
}
