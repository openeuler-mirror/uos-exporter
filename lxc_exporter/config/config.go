package config

import (
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"lxc_exporter/pkg/utils"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure: false,
	}
)


// TODO: implement
