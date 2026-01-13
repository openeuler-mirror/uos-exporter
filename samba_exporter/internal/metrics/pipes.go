package metrics

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
)

type PipeTypeT string

const requestPipeFileName = "samba_exporter.request.pipe"
const responsePipeFileName = "samba_exporter.response.pipe"
const pipePath = "/run"
const testPipePath = "/dev/shm"
const pipePermission = 0660
const endByte byte = 0

const (
	RequestPipe PipeTypeT = "REQUEST_PIPE"
	ResposePipe PipeTypeT = "RESPONSE_PIPE"
)

// PipeHandler - 管道通信处理器类型
type PipeHandler struct {
	TestMode bool
	PipeType PipeTypeT
	mMutext  sync.Mutex
	pipePath string // 缓存管道路径
}

// NewPipeHandler - 创建新的管道处理器实例
func NewPipeHandler(testMode bool, pipeType PipeTypeT) *PipeHandler {
	handler := &PipeHandler{
		TestMode: testMode,
		PipeType: pipeType,
	}
	handler.pipePath = handler.GetPipeFilePath() // 初始化时计算管道路径
	return handler
}

// GetPipeFilePath -  Get the path to the named pipe files for this application
func (handler *PipeHandler) GetPipeFilePath() string {
	var dirname string
	if handler.TestMode {
		dirname = testPipePath
	} else {
		dirname = pipePath
	}

	var pipeFileName string
	if handler.PipeType == RequestPipe {
		pipeFileName = requestPipeFileName
	} else {
		pipeFileName = responsePipeFileName
	}

	return fmt.Sprintf("%s/%s", dirname, pipeFileName)
}

// PipeExists - Check if the named pipe files for this application exists
func (handler *PipeHandler) PipeExists() bool {
	return FileExists(handler.GetPipeFilePath())
}

// WaitForPipeInputBytes - Blocking! Wait for input in the pipe and return it as byte array
// The array will be empty in case of errors
func (handler *PipeHandler) WaitForPipeInputBytes() ([]byte, error) {
	handler.mMutext.Lock()
	defer handler.mMutext.Unlock()

	reader, errGet := handler.getReaderPipe()
	if errGet != nil {
		logrus.Error(fmt.Sprintf("Get read pipe error %v", errGet))
		return []byte{}, errGet
	}

	received, errRead := reader.ReadBytes(endByte)
	if errRead != nil {
		if errRead != io.EOF {
			return []byte{}, errRead
		}
		return received, nil
	}

	return received[0 : len(received)-1], nil
}

// WaitForPipeInputString - Blocking! Wait for input in the pipe and return it as string
// The string will be empty in case of errors
func (handler *PipeHandler) WaitForPipeInputString() (string, error) {
	data, err := handler.WaitForPipeInputBytes()

	return strings.TrimSpace(string(data)), err
}

// ensurePipeExists - 确保管道存在
func (handler *PipeHandler) ensurePipeExists() error {
	if handler.PipeExists() {
		return nil
	}
	return syscall.Mkfifo(handler.pipePath, pipePermission)
}

// WritePipeBytes - Write byte data to the pipe
func (handler *PipeHandler) WritePipeBytes(data []byte) error {
	handler.mMutext.Lock()
	defer handler.mMutext.Unlock()

	writer, errGet := handler.getWriterPipe()
	if errGet != nil {
		return errGet
	}
	data = append(data, endByte)
	_, errWrite := writer.Write(data)
	if errWrite != nil {
		return errWrite
	}
	errFlush := writer.Flush()
	if errFlush != nil {
		return errFlush
	}

	return nil
}

// WritePipeString - Write string data to the pipe
func (handler *PipeHandler) WritePipeString(data string) error {
	return handler.WritePipeBytes([]byte(data))
}

// FileExists - Check if a file exists. Return false in case the path does not exist or is a directory
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}

	if info.IsDir() {
		return false
	}
	return true
}

// GetReaderPipe - Get a new reader for the common pipe.
//
//	Remember: This is a blocking call and will return once data can be read from the pipe
func (handler *PipeHandler) getReaderPipe() (*bufio.Reader, error) {
	if err := handler.ensurePipeExists(); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(handler.pipePath, os.O_RDONLY|os.O_CREATE, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("打开管道文件失败: %w", err)
	}
	return bufio.NewReader(file), nil

}

// GetWriterPipe - Get a new writer for the common pipe.
func (handler *PipeHandler) getWriterPipe() (*bufio.Writer, error) {
	if err := handler.ensurePipeExists(); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(handler.pipePath, os.O_RDWR|os.O_CREATE, pipePermission)
	if err != nil {
		return nil, fmt.Errorf("打开管道文件失败: %w", err)
	}
	return bufio.NewWriter(file), nil
}

func (handler *PipeHandler) createPipe() error {
	return syscall.Mkfifo(handler.GetPipeFilePath(), pipePermission)
}
