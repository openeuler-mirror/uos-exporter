package metrics

import (
	"strings"
	"testing"
)

func TestSmbStatusTimeOutError(t *testing.T) {
	path := "/some/sample/path"
	err := NewSmbStatusTimeOutError(RequestType(path))

	if string(err.Request) != path {
		t.Errorf("The File was %s, but %s was expected", err.Request, path)
	}

	if strings.Contains(err.Error(), path) == false {
		t.Errorf("The error message of SmbStatusTimeOutError does not contain the expected request")
	}
}

func TestSmbStatusUnexpectedResponseError(t *testing.T) {
	path := "/some/sample/path"
	err := NewSmbStatusUnexpectedResponseError(path)

	if string(err.Response) != path {
		t.Errorf("The File was %s, but %s was expected", err.Response, path)
	}

	if strings.Contains(err.Error(), path) == false {
		t.Errorf("The error message of SmbStatusUnexpectedResponseError does not contain the expected request")
	}
}
