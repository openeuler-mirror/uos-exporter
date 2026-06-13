package metrics

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

var createLoggerTests = []struct {
	name                string
	verbose             bool
	file                string
	expectFileContent   string
	expectBufferContent string
}{
	{"default", false, "default", "default", ""},
	{"verbose", true, "verbose", "verbose", "verbose"},
	{"noFile", true, "", "", "noFile"},
}


// TODO: implement functions
