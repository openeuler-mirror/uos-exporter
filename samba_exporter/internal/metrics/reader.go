package metrics

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Type to represent a entry in the 'smbstatus -L -n' output table
type LockData struct {
	PID           int
	ClusterNodeId int // In case smaba is running in cluster mode, otherwise -1
	UserID        int
	DenyMode      string
	Access        string
	AccessMode    string
	Oplock        string
	SharePath     string
	Name          string
	Time          time.Time
}

// Implement Stringer Interface for LockData
func (lockData LockData) String() string {
	if lockData.ClusterNodeId > -1 {
		return fmt.Sprintf("ClusterNodeId: %d; PID: %d; UserID: %d; DenyMode: %s; Access: %s; AccessMode: %s; Oplock: %s; SharePath: %s; Name: %s: Time %s;",
			lockData.ClusterNodeId, lockData.PID, lockData.UserID, lockData.DenyMode, lockData.Access, lockData.AccessMode, lockData.Oplock,
			lockData.SharePath, lockData.Name, lockData.Time.Format(time.RFC3339))
	}
	return fmt.Sprintf("PID: %d; UserID: %d; DenyMode: %s; Access: %s; AccessMode: %s; Oplock: %s; SharePath: %s; Name: %s: Time %s;",
		lockData.PID, lockData.UserID, lockData.DenyMode, lockData.Access, lockData.AccessMode, lockData.Oplock,
		lockData.SharePath, lockData.Name, lockData.Time.Format(time.RFC3339))
}

// GetLockData - Get the entries out of the 'smbstatus -L -n' output table multiline string
// Will return an empty array if the data is in unexpected format
func GetLockData(data string) []LockData {
	if shouldSkipLockData(data) {
		return nil
	}

	lines := strings.Split(data, "\n")
	sepIndex := findSeperatorLineIndex(lines)
	if !isValidLockHeader(lines, sepIndex) {
		return nil
	}

	var results []LockData
	for i, line := range lines[sepIndex+1:] {
		fields := getFieldMatrix([]string{line}, " ")
		if len(fields) == 0 {
			continue
		}

		entry, err := parseLockEntry(fields[0], line, i)
		if err != nil {
			logrus.Error(err)
			continue
		}
		results = append(results, entry)
	}
	return results
}

// 辅助函数实现
func shouldSkipLockData(data string) bool {
	data = strings.TrimSpace(data)
	return data == "" || strings.HasPrefix(data, NO_LOCKED_FILES)
}

func isValidLockHeader(lines []string, sepIndex int) bool {
	if sepIndex < 1 {
		return false
	}
	matrix := getFieldMatrixFixLength(lines[sepIndex-1:sepIndex], "  ", 9)
	return len(matrix) == 1 && matrix[0][0] == "Pid" && matrix[0][5] == "Oplock"
}

func parseLockEntry(fields []string, line string, lineNum int) (LockData, error) {
	var entry LockData
	var err error

	if err = parsePIDFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("PID parse error: %v", err)
	}

	if err = parseUserID(fields, &entry); err != nil {
		return entry, fmt.Errorf("UserID parse error: %v", err)
	}

	parseLockFields(fields, &entry)

	if entry.Time, err = parseTimeStamp(fields); err != nil {
		return entry, fmt.Errorf("timestamp parse error: %v", err)
	}

	if entry.Name, err = parseFileName(fields); err != nil {
		return entry, fmt.Errorf("filename parse error: %v", err)
	}

	return entry, nil
}

func parsePIDFields(fields []string, entry *LockData) error {
	var err error
	if strings.Contains(fields[0], ":") {
		pidFields := strings.Split(fields[0], ":")
		entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
		if err != nil {
			return err
		}
		entry.PID, err = strconv.Atoi(pidFields[1])
	} else {
		entry.ClusterNodeId = -1
		entry.PID, err = strconv.Atoi(fields[0])
	}
	return err
}

func parseUserID(fields []string, entry *LockData) error {
	var err error
	entry.UserID, err = strconv.Atoi(fields[1])
	return err
}

func parseLockFields(fields []string, entry *LockData) {
	entry.DenyMode = fields[2]
	entry.Access = fields[3]
	entry.AccessMode = fields[4]
	entry.Oplock = fields[5]
	entry.SharePath = fields[6]
}

func parseTimeStamp(fields []string) (time.Time, error) {
	fieldLength := len(fields)
	timeConvSuc, connectTime := tryGetTimeStampFromStrArr(fields[fieldLength-5 : fieldLength])
	if timeConvSuc {
		return connectTime, nil
	}
	timeConvSuc, connectTime = tryGetTimeStampFromStrArr(fields[fieldLength-6 : fieldLength])
	if timeConvSuc {
		return connectTime, nil
	}
	return time.Time{}, fmt.Errorf("unable to parse timestamp")
}

func parseFileName(fields []string) (string, error) {
	fieldLength := len(fields)
	lastNameIndex := -1

	// 确定时间戳结束位置
	if _, err := parseTimeStamp(fields); err == nil {
		if fieldLength >= 5 {
			lastNameIndex = fieldLength - 5
		}
	} else if fieldLength >= 6 {
		lastNameIndex = fieldLength - 6
	}

	if lastNameIndex == -1 || lastNameIndex <= 7 {
		return "", fmt.Errorf("invalid name field position")
	}

	name := strings.Join(fields[7:lastNameIndex], " ")
	return strings.TrimSpace(name), nil
}

// Type to represent a entry in the 'smbstatus -S -n' output table
type ShareData struct {
	Service       string
	PID           int
	ClusterNodeId int // In case smaba is running in cluster mode, otherwise -1
	Machine       string
	ConnectedAt   time.Time
	Encryption    string
	Signing       string
}

// Implement Stringer Interface for ShareData
func (shareData ShareData) String() string {
	if shareData.ClusterNodeId > -1 {
		return fmt.Sprintf("Service: %s; ClusterNodeId: %d; PID: %d; Machine: %s; ConnectedAt: %s; Encryption: %s; Signing: %s;",
			shareData.Service, shareData.ClusterNodeId, shareData.PID, shareData.Machine, shareData.ConnectedAt.Format(time.RFC3339),
			shareData.Encryption, shareData.Signing)
	}
	return fmt.Sprintf("Service: %s; PID: %d; Machine: %s; ConnectedAt: %s; Encryption: %s; Signing: %s;",
		shareData.Service, shareData.PID, shareData.Machine, shareData.ConnectedAt.Format(time.RFC3339),
		shareData.Encryption, shareData.Signing)
}

// GetShareData - Get the entries out of the 'smbstatus -S -n' output table multiline string
// Will return an empty array if the data is in unexpected format
func GetShareData(data string) []ShareData {
	if shouldSkipShareData(data) {
		return nil
	}

	lines := strings.Split(data, "\n")
	sepIndex := findSeperatorLineIndex(lines)
	if !isValidShareHeader(lines, sepIndex) {
		return nil
	}

	runningMode := detectRunningMode(lines[sepIndex-1 : sepIndex])
	switch runningMode {
	case "normal":
		return parseNormalShareData(lines[sepIndex+1:])
	case "cluster":
		return parseClusterShareData(lines[sepIndex+1:])
	default:
		return nil
	}
}

// 辅助函数实现
func shouldSkipShareData(data string) bool {
	data = strings.TrimSpace(data)
	return data == ""
}

func isValidShareHeader(lines []string, sepIndex int) bool {
	if sepIndex < 1 {
		return false
	}
	return true
}

func detectRunningMode(headerLines []string) string {
	// 普通模式检查
	matrix6 := getFieldMatrixFixLength(headerLines, "  ", 6)
	if len(matrix6) == 1 && matrix6[0][0] == "Service" && matrix6[0][3] == "Connected at" {
		return "normal"
	}

	// 集群模式检查
	matrix7 := getFieldMatrixFixLength(headerLines, "  ", 7)
	if len(matrix7) == 1 && matrix7[0][0] == "PID" && matrix7[0][4] == "Protocol Version" {
		return "cluster"
	}

	return "none"
}

func parseNormalShareData(lines []string) []ShareData {
	var results []ShareData
	for i, line := range lines {
		fields := getFieldMatrix([]string{line}, " ")
		if len(fields) == 0 {
			continue
		}

		entry, err := parseNormalShareEntry(fields[0], line, i)
		if err != nil {
			logrus.Error(err)
			continue
		}
		results = append(results, entry)
	}
	return results
}

func parseClusterShareData(lines []string) []ShareData {
	var results []ShareData
	for i, line := range lines {
		fields := getFieldMatrix([]string{line}, " ")
		if len(fields) == 0 {
			continue
		}

		entry, err := parseClusterShareEntry(fields[0], line, i)
		if err != nil {
			logrus.Error(err)
			continue
		}
		results = append(results, entry)
	}
	return results
}

func parseNormalShareEntry(fields []string, line string, lineNum int) (ShareData, error) {
	var entry ShareData
	var err error

	if err = parseServiceAndPID(fields, &entry); err != nil {
		return entry, fmt.Errorf("service/PID parse error: %v", err)
	}

	if err = parseMachineAndTime(fields, &entry); err != nil {
		return entry, fmt.Errorf("machine/time parse error: %v", err)
	}

	if err = parseEncryptionFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("encryption fields parse error: %v", err)
	}

	return entry, nil
}

func parseClusterShareEntry(fields []string, line string, lineNum int) (ShareData, error) {
	var entry ShareData
	var err error

	if err = parseClusterPIDFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("cluster PID parse error: %v", err)
	}

	if err = parseClusterMachineFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("cluster machine parse error: %v", err)
	}

	if err = parseClusterEncryptionFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("cluster encryption parse error: %v", err)
	}

	return entry, nil
}

func parseServiceAndPID(fields []string, entry *ShareData) error {
	var err error
	fieldLength := len(fields)

	if strings.Contains(fields[1], ":") {
		pidFields := strings.Split(fields[1], ":")
		entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
		if err != nil {
			return err
		}
		entry.PID, err = strconv.Atoi(pidFields[1])
	} else {
		entry.ClusterNodeId = -1
		lastNameField := -1
		pidFound := true

		for {
			lastNameField++
			entry.PID, err = strconv.Atoi(fields[lastNameField+1])
			if err == nil {
				break
			}
			if fieldLength-11 <= lastNameField {
				pidFound = false
				break
			}
		}

		if !pidFound {
			return fmt.Errorf("failed to parse PID")
		}
		entry.Service = strings.Join(fields[0:lastNameField+1], " ")
	}
	return nil
}

func parseMachineAndTime(fields []string, entry *ShareData) error {
	fieldLength := len(fields)
	lastNameField := -1

	// 确定PID字段位置
	if strings.Contains(fields[1], ":") {
		lastNameField = 1
	} else {
		for i := 1; i < fieldLength; i++ {
			if _, err := strconv.Atoi(fields[i]); err == nil {
				lastNameField = i
				break
			}
		}
	}

	if lastNameField == -1 {
		return fmt.Errorf("failed to determine PID field position")
	}

	entry.Machine = fields[lastNameField+1]

	// 解析时间戳
	// logrus.Infof("trying to parse timestamp%s", fields[lastNameField+2:fieldLength-2])
	timeConvSuc, connectTime := tryGetTimeStampFromStrArr(fields[lastNameField+2 : fieldLength-2])
	if !timeConvSuc {
		return fmt.Errorf("failed to parse timestamp")
	}
	entry.ConnectedAt = connectTime

	return nil
}

func parseEncryptionFields(fields []string, entry *ShareData) error {
	fieldLength := len(fields)
	lastNameField := -1

	// 确定PID字段位置
	if strings.Contains(fields[1], ":") {
		lastNameField = 1
	} else {
		for i := 1; i < fieldLength; i++ {
			if _, err := strconv.Atoi(fields[i]); err == nil {
				lastNameField = i
				break
			}
		}
	}

	if lastNameField == -1 {
		return fmt.Errorf("failed to determine PID field position")
	}

	// 确定时间戳结束位置
	timeIndex := -1
	if err := parseMachineAndTime(fields, entry); err == nil {
		timeIndex = lastNameField + 2 + 5 // 假设时间戳占5个字段
	}

	if timeIndex == -1 || timeIndex >= fieldLength-2 {
		return fmt.Errorf("failed to determine encryption fields position")
	}

	entry.Encryption = fields[timeIndex+2]
	entry.Signing = fields[timeIndex+2]

	return nil
}

func parseClusterPIDFields(fields []string, entry *ShareData) error {
	var err error

	if strings.Contains(fields[0], ":") {
		pidFields := strings.Split(fields[0], ":")
		entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
		if err != nil {
			return err
		}
		entry.PID, err = strconv.Atoi(pidFields[1])
	} else {
		entry.ClusterNodeId = -1
		entry.PID, err = strconv.Atoi(fields[0])
	}

	return err
}

func parseClusterMachineFields(fields []string, entry *ShareData) error {
	fieldLength := len(fields)

	if fieldLength == 8 {
		entry.Machine = fmt.Sprintf("%s %s", fields[3], fields[4])
	} else if fieldLength == 7 {
		entry.Machine = fields[3]
	} else {
		return fmt.Errorf("invalid field length for cluster mode")
	}

	return nil
}

func parseClusterEncryptionFields(fields []string, entry *ShareData) error {
	fieldLength := len(fields)

	if fieldLength == 8 {
		entry.Encryption = fields[6]
		entry.Signing = fields[7]
	} else if fieldLength == 7 {
		entry.Encryption = fields[5]
		entry.Signing = fields[6]
	} else {
		return fmt.Errorf("invalid field length for cluster mode")
	}

	return nil
}

// Type to represent a entry in the 'smbstatus -p -n' output table
type ProcessData struct {
	PID             int
	ClusterNodeId   int // In case smaba is running in cluster mode, otherwise -1
	UserID          int
	GroupID         int
	Machine         string
	ProtocolVersion string
	Encryption      string
	Signing         string
	SambaVersion    string
}

// Implement Stringer Interface for ProcessData
func (processData ProcessData) String() string {
	if processData.ClusterNodeId > -1 {
		return fmt.Sprintf("ClusterNodeId: %d; PID: %d; UserID: %d; GroupID: %d; Machine: %s; ProtocolVersion: %s; Encryption: %s; Signing: %s;",
			processData.ClusterNodeId, processData.PID, processData.UserID, processData.GroupID, processData.Machine, processData.ProtocolVersion,
			processData.Encryption, processData.Signing)
	}
	return fmt.Sprintf("PID: %d; UserID: %d; GroupID: %d; Machine: %s; ProtocolVersion: %s; Encryption: %s; Signing: %s;",
		processData.PID, processData.UserID, processData.GroupID, processData.Machine, processData.ProtocolVersion,
		processData.Encryption, processData.Signing)
}

// GetProcessData - Get the entries out of the 'smbstatus -p -n' output table multiline string
// Will return an empty array if the data is in unexpected format
func GetProcessData(data string) []ProcessData {
	if shouldSkipProcessData(data) {
		return nil
	}

	lines := strings.Split(data, "\n")
	sepIndex := findSeperatorLineIndex(lines)
	if !isValidProcessHeader(lines, sepIndex) {
		return nil
	}

	sambaVersion, err := parseSambaVersion(lines[sepIndex-2 : sepIndex-1])
	if err != nil {
		logrus.Error(err)
		return nil
	}

	var results []ProcessData
	for _, line := range lines[sepIndex+1:] {
		fields := getFieldMatrix([]string{line}, " ")
		if len(fields) == 0 {
			continue
		}

		entry, err := parseProcessEntry(fields[0], sambaVersion)
		if err != nil {
			logrus.WithField("line", line).Error(err)
			continue
		}
		results = append(results, entry)
	}
	return results
}

// 辅助函数实现
func shouldSkipProcessData(data string) bool {
	data = strings.TrimSpace(data)
	return data == ""
}

func isValidProcessHeader(lines []string, sepIndex int) bool {
	if sepIndex < 2 {
		return false
	}
	matrix := getFieldMatrixFixLength(lines[sepIndex-1:sepIndex], "  ", 7)
	return len(matrix) == 1 && matrix[0][1] == "Username" && matrix[0][4] == "Protocol Version"
}

func parseSambaVersion(versionLines []string) (string, error) {
	versionLine := versionLines[0]
	if !strings.HasPrefix(versionLine, "Samba version") {
		return "", fmt.Errorf("invalid Samba version line format")
	}
	return strings.TrimSpace(strings.Replace(versionLine, "Samba version", "", 1)), nil
}

func parseProcessEntry(fields []string, sambaVersion string) (ProcessData, error) {
	var entry ProcessData
	var err error

	if err = parseProcessPIDFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("PID parse error: %v", err)
	}

	if err = parseUserAndGroupFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("user/group parse error: %v", err)
	}

	if err = parseMachineAndProtocolFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("machine/protocol parse error: %v", err)
	}

	if err = parseSecurityFields(fields, &entry); err != nil {
		return entry, fmt.Errorf("security fields parse error: %v", err)
	}

	entry.SambaVersion = sambaVersion
	return entry, nil
}

func parseProcessPIDFields(fields []string, entry *ProcessData) error {
	var err error
	if strings.Contains(fields[0], ":") {
		pidFields := strings.Split(fields[0], ":")
		entry.ClusterNodeId, err = strconv.Atoi(pidFields[0])
		if err != nil {
			return err
		}
		entry.PID, err = strconv.Atoi(pidFields[1])
	} else {
		entry.ClusterNodeId = -1
		entry.PID, err = strconv.Atoi(fields[0])
	}
	return err
}

func parseUserAndGroupFields(fields []string, entry *ProcessData) error {
	var err error

	// 处理用户ID
	if fields[1] == "nobody" {
		entry.UserID = -1
	} else {
		entry.UserID, err = strconv.Atoi(fields[1])
		if err != nil {
			return err
		}
	}

	// 处理组ID
	if fields[2] == "nogroup" {
		entry.GroupID = -1
	} else {
		entry.GroupID, err = strconv.Atoi(fields[2])
		if err != nil {
			return err
		}
	}

	return nil
}

func parseMachineAndProtocolFields(fields []string, entry *ProcessData) error {
	fieldLength := len(fields)

	if fieldLength == 8 {
		entry.Machine = fmt.Sprintf("%s %s", fields[3], fields[4])
		entry.ProtocolVersion = fields[5]
	} else if fieldLength == 7 {
		entry.Machine = fields[3]
		entry.ProtocolVersion = fields[4]
	} else {
		return fmt.Errorf("invalid field length %d", fieldLength)
	}

	return nil
}

func parseSecurityFields(fields []string, entry *ProcessData) error {
	fieldLength := len(fields)

	if fieldLength == 8 {
		entry.Encryption = fields[6]
		entry.Signing = fields[7]
	} else if fieldLength == 7 {
		entry.Encryption = fields[5]
		entry.Signing = fields[6]
	} else {
		return fmt.Errorf("invalid field length %d", fieldLength)
	}

	return nil
}

func GetPsData(data string) []PsUtilPidData {
	var ret []PsUtilPidData
	errConv := json.Unmarshal([]byte(data), &ret)
	if errConv != nil {
		logrus.Error(errConv, "while converting PsData json")
		return []PsUtilPidData{}
	}

	return ret
}

func getFieldMatrixFixLength(dataLines []string, separator string, lineFields int) [][]string {

	var fieldMatrix [][]string

	for _, matrixLine := range getFieldMatrix(dataLines, separator) {
		if len(matrixLine) == lineFields {
			fieldMatrix = append(fieldMatrix, matrixLine)
		}
	}

	return fieldMatrix
}

func getFieldMatrix(dataLines []string, separator string) [][]string {

	var fieldMatrix [][]string

	for _, line := range dataLines {
		fields := strings.Split(line, separator)
		var matrixLine []string
		for _, field := range fields {
			trimmedField := strings.TrimSpace(field)
			if trimmedField != "" {
				matrixLine = append(matrixLine, trimmedField)
			}
		}
		fieldMatrix = append(fieldMatrix, matrixLine)
	}

	return fieldMatrix
}

func concatStrFromArr(fields []string) string {
	ret := ""
	for i, field := range fields {
		if i == 0 {
			ret = field
		} else {
			ret = ret + " " + field
		}
	}

	return ret
}

func tryGetTimeStampFromStrArr(fields []string) (bool, time.Time) {
	timeStr := ""
	var ret time.Time
	var err error
	for _, sec := range fields {
		timeStr = fmt.Sprintf("%s %s", timeStr, sec)
	}
	timeStr = strings.TrimSpace(timeStr)
	ret, err = time.ParseInLocation(time.ANSIC, timeStr, time.Now().Location())
	if err == nil {
		return true, ret
	}

	ret, err = time.Parse(time.ANSIC, timeStr)
	if err == nil {
		return true, ret
	}

	ret, err = time.Parse("Mon Jan 02 03:04:05 PM 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse("Mon Jan 2 03:04:05 PM 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}

	ret, err = time.Parse("Mon Jan _2 15:04:05 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}
	ret, err = time.Parse("Mo Jan _2 15:04:05 2006 MST", timeStr)
	if err == nil {
		return true, ret
	}

	// logrus.Infof("tryGetTimeStampFromStrArr: %v", err)
	return false, time.Now()
}

func findSeperatorLineIndex(lines []string) int {

	for i, line := range lines {
		if strings.HasPrefix(line, "-----------------------------------------") {
			return i
		}
	}

	return -1
}
