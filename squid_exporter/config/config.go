package config

import (
	"squid_exporter/pkg/utils"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	SquidHostname   *string
	SquidPort       *int
	Login           *string
	Password        *string
	ExtractTimes    *bool
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure: false,
		SquidHostname: "localhost",
		SquidPort:     3128,
		Login:         "",
		Password:      "",
		ExtractTimes:  true,
	}
)


// TODO: implement functions
