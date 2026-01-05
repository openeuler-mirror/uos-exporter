package crmmon

import (
	"encoding/xml"
	"hacluster_exporter/pkg/utils"

	"github.com/pkg/errors"
)

type Parser interface {
	Parse() (Root, error)
}

type crmMonParser struct {
	crmMonPath string
}

func (c *crmMonParser) Parse() (crmMon Root, err error) {
	crmMonXML, err := utils.RunCommand(c.crmMonPath, "-X", "--inactive")
	if err != nil {
		return crmMon, errors.Wrap(err, "error while executing crm_mon")
	}

	err = xml.Unmarshal(crmMonXML, &crmMon)
	if err != nil {
		return crmMon, errors.Wrap(err, "error while parsing crm_mon XML output")
	}

	return crmMon, nil
}

func NewCrmMonParser(crmMonPath string) *crmMonParser {
	return &crmMonParser{crmMonPath}
}
