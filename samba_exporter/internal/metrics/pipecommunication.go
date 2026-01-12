package metrics

// Copyright 2021 by tobi@backfrak.de. All
// rights reserved. Use of this source code is governed
// by a BSD-style license that can be found in the
// LICENSE file.

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var requestCount = 0
var requestMux sync.Mutex
var collectMux sync.Mutex

type smbResponse struct {
	Data  string
	Error error
}

// GetSambaStatus - 从samba_statusd获取所有数据表输出
func GetSambaStatus(requestHandler *PipeHandler, responseHandler *PipeHandler, requestTimeOut int) ([]LockData, []ProcessData, []ShareData, []PsUtilPidData, error) {
	collectMux.Lock()
	defer collectMux.Unlock()

	// 并行获取各种数据
	locksChan := make(chan []LockData, 1)
	processesChan := make(chan []ProcessData, 1)
	sharesChan := make(chan []ShareData, 1)
	psdataChan := make(chan []PsUtilPidData, 1)
	errinfo := make(chan error, 1)

	// var wg sync.WaitGroup
	// wg.Add(4)

	go func() {
		// defer wg.Done()
		res, err := getSmbStatusDataTimeOut(requestHandler, responseHandler, LOCK_REQUEST, requestTimeOut)
		if err == nil {
			locksChan <- GetLockData(res)
			errinfo <- nil
		} else {
			locksChan <- nil
			errinfo <- err
		}
	}()

	go func() {
		// defer wg.Done()
		res, err := getSmbStatusDataTimeOut(requestHandler, responseHandler, PROCESS_REQUEST, requestTimeOut)
		if err == nil {
			processesChan <- GetProcessData(res)
			errinfo <- nil
		} else {
			processesChan <- nil
			errinfo <- err
		}
	}()

	go func() {
		// defer wg.Done()
		res, err := getSmbStatusDataTimeOut(requestHandler, responseHandler, SHARE_REQUEST, requestTimeOut)
		if err == nil {
			sharesChan <- GetShareData(res)
			errinfo <- nil
		} else {
			sharesChan <- nil
			errinfo <- err
		}
	}()

	go func() {
		// defer wg.Done()
		res, err := getSmbStatusDataTimeOut(requestHandler, responseHandler, PS_REQUEST, requestTimeOut)
		if err == nil {
			psdataChan <- GetPsData(res)
			errinfo <- nil
		} else {
			psdataChan <- nil
			errinfo <- err
		}
	}()

	// wg.Wait()

	locks := <-locksChan
	processes := <-processesChan
	shares := <-sharesChan
	psdata := <-psdataChan
	err := <-errinfo

	// 检查空数据情况
	if len(shares) == 0 {
		logrus.Info("Got an empty share table when requesting \"smbstatus -S -n\" from samba_statusd")
	}
	if len(processes) == 0 {
		logrus.Info("Got an empty process table when requesting \"smbstatus -p -n\" from samba_statusd")
	}
	if len(locks) == 0 {
		logrus.Info("Got an empty locks table when requesting \"smbstatus -L -n\" from samba_statusd")
	}
	if len(psdata) == 0 {
		logrus.Info("Got an empty psdata table when requesting")
	}
	// logrus.Info("Got all status data")
	return locks, processes, shares, psdata, err
}

func goGetProcessData(res string, c chan []ProcessData) {
	processes := GetProcessData(res)

	c <- processes
}

func goGetShareData(res string, c chan []ShareData) {
	shares := GetShareData(res)

	c <- shares
}

func goGetLockData(res string, c chan []LockData) {
	locks := GetLockData(res)

	c <- locks
}

func goGetPsData(res string, c chan []PsUtilPidData) {
	locks := GetPsData(res)

	c <- locks
}

func getSmbStatusDataTimeOut(requestHandler *PipeHandler, responseHandler *PipeHandler, request RequestType, requestTimeOut int) (string, error) {
	resultChan := make(chan smbResponse, 1)

	go func() {
		goGetSmbStatusData(requestHandler, responseHandler, request, resultChan)
	}()

	// logrus.Info(fmt.Sprintf("Wait for \"%s\" response", request))

	select {
	case res := <-resultChan:
		if res.Error != nil {
			return "", res.Error
		}
		logrus.Info(fmt.Sprintf("Get \"%s\" response", request))
		return res.Data, nil
	case <-time.After(time.Second * time.Duration(requestTimeOut)):
		if err := requestHandler.WritePipeString(""); err != nil {
			logrus.Errorf("Clear request pipe failed: %v", err)
		}
		return "", NewSmbStatusTimeOutError(request)
	}
}

func goGetSmbStatusData(requestHandler *PipeHandler, responseHandler *PipeHandler, request RequestType, c chan smbResponse) {
	retStr, err := getSmbStatusData(requestHandler, responseHandler, request)

	ret := smbResponse{retStr, err}

	c <- ret
}

// getSmbStatusData - 获取samba状态数据
func getSmbStatusData(requestHandler *PipeHandler, responseHandler *PipeHandler, request RequestType) (string, error) {
	// Ensure we run only one request per time on the pipes
	requestMux.Lock()
	defer requestMux.Unlock()
	requestCount++
	requestString := GetRequest(request, requestCount)

	logrus.Info(fmt.Sprintf("Send \"%s\" request with ID %d on pipe", request, requestCount))

	errWrite := requestHandler.WritePipeString(requestString)
	if errWrite != nil {
		return "", errWrite
	}

	logrus.Info(fmt.Sprintf("Wait for \"%s\" response with ID %d on pipe", request, requestCount))

	response := ""
	var errRead error

	// as long as the response pipe is empty, wait for response
	for response == "" {
		response, errRead = responseHandler.WaitForPipeInputString()
		if errRead != nil {
			return "", errRead
		}
	}

	logrus.Info(fmt.Sprintf("Handle \"%s\" response with ID %d from pipe", request, requestCount))

	header, data, errSplit := SplitResponse(response)
	if errSplit != nil {
		return "", errSplit
	}

	if !CheckResponseHeader(header, request, requestCount) {
		return "", NewReaderError(response, request, requestCount)
	}

	return data, nil
}

// func getSmbStatusData(requestHandler *PipeHandler, responseHandler *PipeHandler, request RequestType) (string, error) {
// 	requestMux.Lock()
// 	defer requestMux.Unlock()

// 	requestCount++
// 	requestString := GetRequest(request, requestCount)

// 	logrus.Infof("Send \"%s\" request with ID %d on pipe", request, requestCount)
// 	if err := requestHandler.WritePipeString(requestString); err != nil {
// 		return "", err
// 	}

// 	logrus.Infof("Wait for \"%s\" response with ID %d on pipe", request, requestCount)
// 	response, err := responseHandler.WaitForPipeInputString()
// 	if err != nil {
// 		return "", err
// 	}

// 	logrus.Infof("Handle \"%s\" response with ID %d from pipe", request, requestCount)
// 	header, data, err := SplitResponse(response)
// 	if err != nil {
// 		return "", err
// 	}

// 	if !CheckResponseHeader(header, request, requestCount) {
// 		return "", NewReaderError(response, request, requestCount)
// 	}

// 	return data, nil
// }
