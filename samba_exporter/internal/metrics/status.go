package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/sirupsen/logrus"
)

const PROCESS_TO_MONITOR = "smbd"

// Type for functions that can create a response string
type response func(*PipeHandler, int) error

// Path to the smbstatus executable
var smbstatusPath string = "/usr/bin/smbstatus"

var requestQueue StringQueue

var psDataGenerator *PsDataGenerator

func Getstatus() int {
	config, err := LoadConfig("/etc/uos-exporter/samba-exporter.yaml")
	if err != nil {
		logrus.Errorf("Error parse samba config %v\n", err)
	} else {
		parms.Test = config.Test
		parms.Test_pipe_mode = config.TestPipeMode
		parms.Request_timeout = config.RequestTimeOut
		parms.Do_not_expose_encryption = config.DoNotExportEncryption
		parms.Do_not_expose_client = config.DoNotExportClient
		parms.Do_not_expose_user = config.DoNotExportUser
		parms.Do_not_expose_pid = config.DoNotExportPid
		parms.Do_not_expose_share_details = config.DoNotExportShareDetails
	}

	RequestHandler := NewPipeHandler(parms.Test, RequestPipe)
	ResponseHandler := NewPipeHandler(parms.Test, ResposePipe)

	logrus.Info(fmt.Sprintf("Named pipe for requests: %s", RequestHandler.GetPipeFilePath()))
	logrus.Info(fmt.Sprintf("Named pipe for response: %s", ResponseHandler.GetPipeFilePath()))

	if !parms.Test {

		currentUser, errUserGet := user.Current()
		if errUserGet != nil {
			logrus.Error(fmt.Sprintf("Error when trying to get the current user: %s", errUserGet.Error()))
			return -5
		}

		if currentUser.Username != "root" {
			logrus.Error(fmt.Sprintf("The current user %s is not root.", currentUser.Username))
			return -6
		}

		var errLookPath error
		smbstatusPath, errLookPath = exec.LookPath("smbstatus")
		if errLookPath != nil {
			logrus.Error("Can not find \"smbstatus\" executable. Please install the needed package.")
			return -3
		} else {
			logrus.Info(fmt.Sprintf("Use %s to get samba status.", smbstatusPath))
		}

		psDataGeneratorTmp, errNewGen := NewPsDataGenerator(PROCESS_TO_MONITOR)
		if errNewGen != nil {
			logrus.Error(errNewGen)
			return -7
		}
		psDataGenerator = psDataGeneratorTmp
	}

	// Ensure we exit clean on term and kill signals
	go waitforKillSignalAndExit()
	go waitforTermSignalAndExit()

	// Init a queue, to store the requests
	requestQueue = *NewStringQueue()

	// Wait for pipe input and process it in an infinite loop
	logrus.Info(fmt.Sprintf("Started %s, waiting for requests in pipe", os.Args[0]))
	for {
		logrus.Info(fmt.Sprintf("Wait for requests in: %s", RequestHandler.GetPipeFilePath()))

		received, errRecv := RequestHandler.WaitForPipeInputString()
		if errRecv != nil {
			logrus.Error(fmt.Sprintf("Receive this unexpected data from the pipe: %s", errRecv))
			return -1
		}

		// Add request to the queue and process the request in own "thread"
		requestQueue.Push(received)
		go goHandleRequestQueue(ResponseHandler)
	}

}

// goHandleRequestQueue, is called as go routine and processes the "oldest" request in the request Queue
func goHandleRequestQueue(responseHandler *PipeHandler) {
	var err error = nil
	var received string
	received, err = requestQueue.Pull()
	if err != nil {
		logrus.Error(fmt.Sprintf("Got error while reading request from Queue: %s", err))
		os.Exit(-8)
	}

	if received == "" {
		return
	}

	if strings.HasPrefix(received, string(PROCESS_REQUEST)) {
		err = handleRequest(responseHandler, received, PROCESS_REQUEST, processResponse, testProcessResponse)
	} else if strings.HasPrefix(received, string(SHARE_REQUEST)) {
		err = handleRequest(responseHandler, received, SHARE_REQUEST, shareResponse, testShareResponse)
	} else if strings.HasPrefix(received, string(LOCK_REQUEST)) {
		err = handleRequest(responseHandler, received, LOCK_REQUEST, lockResponse, testLockResponse)
	} else if strings.HasPrefix(received, string(PS_REQUEST)) {
		err = handleRequest(responseHandler, received, PS_REQUEST, psResponse, testPsResponse)
	} else {
		logrus.Error(fmt.Sprintf("Can not handle the request: '%s'", received))
	}

	if err != nil {
		logrus.Error(fmt.Sprintf("Handle request \"%s\"\n\n: %s", received, err))
		os.Exit(-2)
	}
}

func handleRequest(handler *PipeHandler, request string, requestType RequestType, productiveFunc response, testFunc response) error {
	id, errConv := GetIdFromRequest(request)
	if errConv != nil {
		return nil // In case we cant find an ID, we simply ingnor the request as any other invalid input
	}
	logrus.Info(fmt.Sprintf("Handle \"%s\" with id %d", requestType, id))

	var writeErr error
	if !parms.Test {
		writeErr = productiveFunc(handler, id)
	} else {
		writeErr = testFunc(handler, id)
	}
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func lockResponse(handler *PipeHandler, id int) error {
	header := GetResponseHeader(LOCK_REQUEST, id)
	data, err := exec.Command(smbstatusPath, "-L", "-n").Output()
	if err != nil {
		logrus.Error(fmt.Sprintf("\"%s -L -n\"  returned the following error: %s", smbstatusPath, err))
		os.Exit(-4)
	}
	response := GetResponse(header, string(data))

	return handler.WritePipeString(response)
}

func shareResponse(handler *PipeHandler, id int) error {
	header := GetResponseHeader(SHARE_REQUEST, id)
	data, err := exec.Command(smbstatusPath, "-S", "-n").Output()
	if err != nil {
		logrus.Error(fmt.Sprintf("\"%s -S -n\"  returned the following error: %s", smbstatusPath, err))
		os.Exit(-4)
	}
	response := GetResponse(header, string(data))

	return handler.WritePipeString(response)
}

func processResponse(handler *PipeHandler, id int) error {
	header := GetResponseHeader(PROCESS_REQUEST, id)
	data, err := exec.Command(smbstatusPath, "-p", "-n").Output()
	if err != nil {
		logrus.Error(fmt.Sprintf("\"%s -p -n\"  returned the following error: %s", smbstatusPath, err))
		os.Exit(-4)
	}
	response := GetResponse(header, string(data))

	return handler.WritePipeString(response)
}

func psResponse(handler *PipeHandler, id int) error {
	header := GetResponseHeader(PS_REQUEST, id)
	pidData, err := psDataGenerator.GetPsUtilPidData()
	if err != nil {
		logrus.Error(fmt.Sprintf("\"%s -p -n\"  returned the following error: %s", smbstatusPath, err))
		os.Exit(-4)
	}
	jsonData, errConv := json.MarshalIndent(pidData, "", " ")
	if errConv != nil {
		return errConv
	}
	response := GetResponse(header, string(jsonData))

	return handler.WritePipeString(response)
}

func testPsResponse(handler *PipeHandler, id int) error {
	header := GetResponseHeader(PS_REQUEST, id)
	response := GetResponse(header, TestPsResponse())

	return handler.WritePipeString(response)
}

func testProcessResponse(handler *PipeHandler, id int) error {
	header := GetTestResponseHeader(PROCESS_REQUEST, id)
	response := GetResponse(header, TestProcessResponse)

	return handler.WritePipeString(response)
}

func testShareResponse(handler *PipeHandler, id int) error {
	header := GetTestResponseHeader(SHARE_REQUEST, id)
	response := GetResponse(header, TestShareResponse)

	return handler.WritePipeString(response)
}

func testLockResponse(handler *PipeHandler, id int) error {
	header := GetTestResponseHeader(LOCK_REQUEST, id)
	response := GetResponse(header, TestLockResponse)

	return handler.WritePipeString(response)
}
