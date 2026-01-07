package server

import (
	"bytes"
	"net/http"
	"text/template"
)

type LandingPageConfig struct {
	CSS     string
	Name    string
	Links   []LandingPageLinks
	Version string
}

type LandingPageLinks struct {
	Address string
	Text    string
}

type LandingPageHandler struct {
	landingPage []byte
}


// TODO: implement functions
