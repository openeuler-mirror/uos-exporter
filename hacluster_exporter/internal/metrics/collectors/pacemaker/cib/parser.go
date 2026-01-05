package cib

import (
	"encoding/xml"
	"hacluster_exporter/pkg/utils"

	"github.com/pkg/errors"
)

type Parser interface {
	Parse() (Root, error)
}

type cibAdminParser struct {
	cibAdminPath string
}

func (p *cibAdminParser) Parse() (Root, error) {
	var CIB Root
	cibXML, err := utils.RunCommand(p.cibAdminPath, "--query", "--local")
	if err != nil {
		return CIB, errors.Wrap(err, "error while executing cibadmin")
	}

	err = xml.Unmarshal(cibXML, &CIB)
	if err != nil {
		return CIB, errors.Wrap(err, "could not parse cibadmin status from XML")
	}

	return CIB, nil
}


// TODO: implement functions
