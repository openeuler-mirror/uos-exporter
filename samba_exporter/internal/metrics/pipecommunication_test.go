package metrics

import (
	"fmt"
	"os"
	"testing"
)

func TestGetSambaStatusTimeout(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)
	_, _, _, _, err := GetSambaStatus(requestHandler, responseHandler, 2)

	if err == nil {
		t.Errorf("Exptected an error but got none")
	}

	switch err.(type) {
	case *SmbStatusTimeOutError:
		fmt.Fprintln(os.Stdout, "OK")
	default:
		t.Errorf("Got error '%s' type, but expected '*SmbStatusTimeOutError'", err.Error())
	}

}
