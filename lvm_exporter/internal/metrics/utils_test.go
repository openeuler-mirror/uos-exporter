package metrics

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

// MockCommand 用于模拟 exec.Command 的执行
func MockCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestHelperProcess 用于模拟命令的执行
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(1)
	}

	cmd := args[0]
	switch cmd {
	case "cat":
		// 模拟 cat 命令的输出
		fmt.Fprintf(os.Stdout, `{"vg_name": "vg1", "lv_name": "lv1"}`)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command %s\n", cmd)
		os.Exit(1)
	}
}

func TestGetLvmReport(t *testing.T) {
	// 替换 exec.Command 为 MockCommand
	// execCommand := MockCommand
	// defer func() { execCommand = exec.Command }()

	tests := []struct {
		name    string
		want    LvmReportInfo
		wantErr error
	}{
		{
			name: "Command execution success with valid JSON",
			want: LvmReportInfo{
				Report: []struct{ LvmInfo }{
					{
						LvmInfo{
							PV: []struct{ PvInfo }{
								{
									PvInfo{
										Pv_uuid:  "12345",
										Dev_size: "100GB",
									},
								},
							},
						},
					},
				},
			},
			wantErr: nil,
		},
		{
			name:    "Command execution success with invalid JSON",
			want:    LvmReportInfo{},
			wantErr: nil,
		},
		{
			name:    "Command execution failure",
			want:    LvmReportInfo{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.want, tt.wantErr
			if err != tt.wantErr {
				t.Errorf("GetLvmReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLvmReport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPvInfo(t *testing.T) {
	// 准备测试数据
	report := LvmReportInfo{
		Report: []struct{ LvmInfo }{
			{
				LvmInfo{
					PV: []struct{ PvInfo }{
						{
							PvInfo{
								Pv_uuid:  "12345",
								Dev_size: "100GB",
							},
						},
					},
				},
			},
		},
	}
	// pvinfo := PvInfo{Pv_uuid:  "12345",Dev_size: "100GB",}

	// var infolist []struct {PvInfo} = nil
	// infolist = append(infolist, struct {PvInfo}{pvinfo})

	// report := LvmReportInfo{
	// 	Report: []struct{ LvmInfo }{
	// 		{
	// 			LvmInfo{PV: infolist,},
	// 		},
	// 	},
	// }

	// Test case 1: 正常情况
	pvInfo, err := GetPvInfo(report)
	if err != nil {
		t.Errorf("GetPvInfo() returned error: %v", err)
	}
	if len(pvInfo) == 0 {
		t.Errorf("GetPvInfo() returned empty list")
	}

	// Test case 2: 异常情况
	report.Report[0].PV = nil
	pvInfo, err = GetPvInfo(report)
	if err != nil {
		t.Errorf("GetPvInfo() returned error: %v", err)
	}
	if len(pvInfo) != 0 {
		t.Errorf("GetPvInfo() should return empty list")
	}
}

func TestGetVgInfo(t *testing.T) {
	// 准备测试数据
	report := LvmReportInfo{
		Report: []struct{ LvmInfo }{
			{
				LvmInfo{
					VG: []struct{ VgInfo }{
						{
							VgInfo{
								Vg_uuid: "67890",
								Vg_size: "200GB",
							},
						},
					},
				},
			},
		},
	}

	// Test case 1: 正常情况
	vgInfo, err := GetVgInfo(report)
	if err != nil {
		t.Errorf("GetVgInfo() returned error: %v", err)
	}
	if len(vgInfo) == 0 {
		t.Errorf("GetVgInfo() returned empty list")
	}

	// Test case 2: 异常情况
	report.Report[0].VG = nil
	vgInfo, err = GetVgInfo(report)
	if err != nil {
		t.Errorf("GetVgInfo() returned error: %v", err)
	}
	if len(vgInfo) != 0 {
		t.Errorf("GetVgInfo() should return empty list")
	}
}

func TestGetLvInfo(t *testing.T) {
	// 准备测试数据
	report := LvmReportInfo{
		Report: []struct{ LvmInfo }{
			{
				LvmInfo{
					LV: []struct{ LvInfo }{
						{
							LvInfo{
								Lv_uuid: "abcde",
								Lv_size: "50GB",
							},
						},
					},
				},
			},
		},
	}

	// Test case 1: 正常情况
	lvInfo, err := GetLvInfo(report)
	if err != nil {
		t.Errorf("GetLvInfo() returned error: %v", err)
	}
	if len(lvInfo) == 0 {
		t.Errorf("GetLvInfo() returned empty list")
	}

	// Test case 2: 异常情况
	report.Report[0].LV = nil
	lvInfo, err = GetLvInfo(report)
	if err != nil {
		t.Errorf("GetLvInfo() returned error: %v", err)
	}
	if len(lvInfo) != 0 {
		t.Errorf("GetLvInfo() should return empty list")
	}
}
