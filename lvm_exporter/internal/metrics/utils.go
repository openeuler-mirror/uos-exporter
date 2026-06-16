package metrics

import (
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	//"github.com/shirou/gopsutil"

	"encoding/json"
	"fmt"
)

func run_cmd(cmdstr string) (string, error) {
	cmd := exec.Command("/bin/bash", "-c", cmdstr)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	return string(out), err
}

func parseSize(sizeStr string) float64 {
	if sizeStr == "" {
		sizeStr = "0"
	}
	// 去除字符串中的空格和`<`符号
	sizeStr = strings.TrimSpace(sizeStr)
	sizeStr = strings.TrimPrefix(sizeStr, "<")

	// 统一转换为小写
	sizeStr = strings.ToLower(sizeStr)

	// 解析容量值和单位
	if strings.HasSuffix(sizeStr, "k") { // KB
		sizeStr = strings.TrimSuffix(sizeStr, "k")
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			log.Printf("Error parsing size '%s': %v\n", sizeStr, err)
			return 0
		}
		return size * 1024 // 转换为byte
	} else if strings.HasSuffix(sizeStr, "g") { // GB
		sizeStr = strings.TrimSuffix(sizeStr, "g")
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			log.Printf("Error parsing size '%s': %v\n", sizeStr, err)
			return 0
		}
		return size * 1024 * 1024 * 1024 // 转换为byte
	} else if strings.HasSuffix(sizeStr, "t") { // TB
		sizeStr = strings.TrimSuffix(sizeStr, "t")
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			log.Printf("Error parsing size '%s': %v\n", sizeStr, err)
			return 0
		}
		return size * 1024 * 1024 * 1024 * 1024 // 转换为byte
	} else if strings.HasSuffix(sizeStr, "m") { // MB
		sizeStr = strings.TrimSuffix(sizeStr, "m")
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			log.Printf("Error parsing size '%s': %v\n", sizeStr, err)
			return 0
		}
		return size * 1024 * 1024 // 转换为byte
	} else {
		// 如果没有单位，默认是bytes
		size, err := strconv.ParseFloat(sizeStr, 64)
		if err != nil {
			log.Printf("Error parsing size '%s': %v\n", sizeStr, err)
			return 0
		}
		return size //转换为byte
	}
}

func parseTime(timeStr string) float64 {
	// 去除字符串中的空格
	timeStr = strings.TrimSpace(timeStr)

	// 统一转换为小写
	timeStr = strings.ToLower(timeStr)

	// 解析时间值和单位
	if strings.HasSuffix(timeStr, "s") { // 秒
		timeStr = strings.TrimSuffix(timeStr, "s")
		time, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			log.Printf("Error parsing time '%s': %v\n", timeStr, err)
			return 0
		}
		return time
	} else if strings.HasSuffix(timeStr, "m") { // 分钟
		timeStr = strings.TrimSuffix(timeStr, "m")
		time, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			log.Printf("Error parsing time '%s': %v\n", timeStr, err)
			return 0
		}
		return time * 60 // 转换为秒
	} else if strings.HasSuffix(timeStr, "h") { // 小时
		timeStr = strings.TrimSuffix(timeStr, "h")
		time, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			log.Printf("Error parsing time '%s': %v\n", timeStr, err)
			return 0
		}
		return time * 3600 // 转换为秒
	} else {
		// 如果没有单位，默认是秒
		time, err := strconv.ParseFloat(timeStr, 64)
		if err != nil {
			log.Printf("Error parsing time '%s': %v\n", timeStr, err)
			return 0
		}
		return time
	}
}

func parseString(Str string) float64 {
	if Str == "" {
		return 0
	}
	trimmedStr := strings.TrimSpace(Str)
	str, err := strconv.ParseFloat(trimmedStr, 64)
	if err != nil {
		log.Println("Error convert string to float:", err)
		return 0
	}
	return str
}

func parseTimeStamp(Str string) float64 {
	if Str == "" {
		return 0
	}
	// 定义时间格式（注意：格式必须与时间字符串一致）
	layout := "2006-01-02 15:04:05 -0700"

	// 解析时间字符串
	t, err := time.Parse(layout, Str)
	if err != nil {
		log.Println("解析时间失败:", err)
		return 0
	}

	// 转换为 Unix 时间戳（秒数）
	// unixSeconds := t.Unix()

	// 转换为 Unix 时间戳（浮点数，包含小数部分）
	unixSecondsFloat := float64(t.Unix()) + float64(t.Nanosecond())/1e9

	return unixSecondsFloat
}

type Row map[string]string

type LvmReportInfo struct {
	Report []struct{ LvmInfo } `json:"report"`
}

type LvmInfo struct {
	PV    []struct{ PvInfo }    `json:"pv,omitempty"`
	LV    []struct{ LvInfo }    `json:"lv,omitempty"`
	VG    []struct{ VgInfo }    `json:"vg,omitempty"`
	SEG   []struct{}            `json:"seg,omitempty"`
	PVSEG []struct{ PvSegInfo } `json:"pvseg,omitempty"`
}

type PvInfo struct {
	Pv_uuid           string `json:"pv_uuid"`
	Dev_size          string `json:"dev_size"`
	Pe_start          string `json:"pe_start"`
	Pv_allocatable    string `json:"pv_allocatable"`
	Pv_ba_size        string `json:"pv_ba_size"`
	Pv_ba_start       string `json:"pv_ba_start"`
	Pv_duplicate      string `json:"pv_duplicate"`
	Pv_exported       string `json:"pv_exported"`
	Pv_ext_vsn        string `json:"pv_ext_vsn"`
	Pv_free           string `json:"pv_free"`
	Pv_in_use         string `json:"pv_in_use"`
	Pv_major          string `json:"pv_major"`
	Pv_minor          string `json:"pv_minor"`
	Pv_mda_count      string `json:"pv_mda_count"`
	Pv_mda_free       string `json:"pv_mda_free"`
	Pv_mda_size       string `json:"pv_mda_size"`
	Pv_mda_used_count string `json:"pv_mda_used_count"`
	Pv_missing        string `json:"pv_missing"`
	Pv_pe_alloc_count string `json:"pv_pe_alloc_count"`
	Pv_pe_count       string `json:"pv_pe_count"`
	Pv_size           string `json:"pv_size"`
	Pv_used           string `json:"pv_used"`
}

type VgInfo struct {
	Vg_uuid             string `json:"vg_uuid"`
	Lv_count            string `json:"lv_count"`
	Max_lv              string `json:"max_lv"`
	Max_pv              string `json:"max_pv"`
	Pv_count            string `json:"pv_count"`
	Snap_count          string `json:"snap_count"`
	Vg_clustered        string `json:"vg_clustered"`
	Vg_exported         string `json:"vg_exported"`
	Vg_extendable       string `json:"vg_extendable"`
	Vg_extent_count     string `json:"vg_extent_count"`
	Vg_extent_size      string `json:"vg_extent_size"`
	Vg_free             string `json:"vg_free"`
	Vg_free_count       string `json:"vg_free_count"`
	Vg_mda_copies       string `json:"vg_mda_copies"`
	Vg_mda_count        string `json:"vg_mda_count"`
	Vg_mda_free         string `json:"vg_mda_free"`
	Vg_mda_size         string `json:"vg_mda_size"`
	Vg_mda_used_count   string `json:"vg_mda_used_count"`
	Vg_missing_pv_count string `json:"vg_missing_pv_count"`
	Vg_partial          string `json:"vg_partial"`
	Vg_seqno            string `json:"vg_seqno"`
	Vg_shared           string `json:"vg_shared"`
	Vg_size             string `json:"vg_size"`
}

type LvInfo struct {
	Lv_uuid                     string `json:"lv_uuid"`
	Cache_dirty_blocks          string `json:"cache_dirty_blocks"`
	Cache_read_hits             string `json:"cache_read_hits"`
	Cache_read_misses           string `json:"cache_read_misses"`
	Cache_total_blocks          string `json:"cache_total_blocks"`
	Cache_used_blocks           string `json:"cache_used_blocks"`
	Cache_write_hits            string `json:"cache_write_hits"`
	Cache_write_misses          string `json:"cache_write_misses"`
	Copy_percent                string `json:"copy_percent"`
	Data_percent                string `json:"data_percent"`
	Integritymismatches         string `json:"interitymismatches"`
	Lv_active_exclusively       string `json:"lv_active_exclusively"`
	Lv_active_locally           string `json:"lv_active_locally"`
	Lv_active_remotely          string `json:"lv_active_remotely"`
	Lv_allocation_locked        string `json:"lv_allocation_locaked"`
	Lv_check_needed             string `json:"lv_check_needed"`
	Lv_converting               string `json:"lv_converting"`
	Lv_device_open              string `json:"lv_device_open"`
	Lv_fixed_minor              string `json:"lv_fixed_minor"`
	Lv_historical               string `json:"lv_historical"`
	Lv_image_synced             string `json:"lv_image_synced"`
	Lv_inactive_table           string `json:"lv_inactive_table"`
	Lv_initial_image_sync       string `json:"lv_initial_image_sync"`
	Lv_kernel_major             string `json:"lv_kernel_major"`
	Lv_kernel_minor             string `json:"lv_kernel_minor"`
	Lv_live_table               string `json:"lv_live_table"`
	Lv_major                    string `json:"lv_major"`
	Lv_merge_failed             string `json:"lv_merge_failed"`
	Lv_merging                  string `json:"lv_merging"`
	Lv_metadata_size            string `json:"lv_metadata_size"`
	Lv_minor                    string `json:"lv_minor"`
	Lv_read_ahead               string `json:"lv_read_ahead"`
	Lv_size                     string `json:"lv_size"`
	Lv_skip_activation          string `json:"lv_skip_activation"`
	Lv_snapshot_invalid         string `json:"lv_snapshot_invalid"`
	Lv_suspended                string `json:"lv_suspended"`
	Lv_time                     string `json:"lv_time"`
	Lv_time_removed             string `json:"lv_time_removed"`
	Metadata_percent            string `json:"lv_metadata_percent"`
	Origin_size                 string `json:"origin_size"`
	Raid_max_recovery_rate      string `json:"raid_max_recovery_rate"`
	Raid_min_recovery_rate      string `json:"raid_min_recovery_rate"`
	Raid_mismatch_count         string `json:"raid_mismatch_count"`
	Raid_write_behind           string `json:"raid_write_behind"`
	Raidintegrityblocksize      string `json:"raidintegrityblocksize"`
	Seg_count                   string `json:"seg_count"`
	Snap_percent                string `json:"snap_percent"`
	Sync_percent                string `json:"sync_percent"`
	Vdo_saving_percent          string `json:"vdo_saving_percent"`
	Vdo_used_size               string `json:"vdo_used_size"`
	Writecache_error            string `json:"writecache_error"`
	Writecache_free_blocks      string `json:"writecache_free_blocks"`
	Writecache_total_blocks     string `json:"writecache_total_blocks"`
	Writecache_writeback_blocks string `json:"writecache_writeback_blocks"`
}

type PvSegInfo struct {
	Pv_uuid     string `json:"pv_uuid"`
	Lv_uuid     string `json:"lv_uuid"`
	Pvseg_start string `json:"pvseg_start"`
	Pvseg_size  string `json:"pvseg_size"`
}

var (
	report LvmReportInfo
)

func GetLvmReport() (LvmReportInfo, error) {
	// 执行LVM fullreport命令
	_, err := run_cmd("lvm fullreport --reportformat json > /tmp/lvmresult.json")
	if err != nil {
		fmt.Println("cmd run failed with ", err)
	}

	cmd := exec.Command("cat", "/tmp/lvmresult.json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error executing LVM command:", err)
		fmt.Println("Output:", string(output))
		return report, err
	}

	// 解析JSON数据
	if err := json.Unmarshal(output, &report); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return report, err
	}

	return report, nil
}

func GetPvInfo(report LvmReportInfo) ([]PvInfo, error) {
	var InfoList []PvInfo = nil

	for _, section := range report.Report {
		for _, pv := range section.PV {
			Info := PvInfo{
				Pv_uuid:           pv.Pv_uuid,
				Dev_size:          pv.Dev_size,
				Pe_start:          pv.Pe_start,
				Pv_allocatable:    pv.Pv_allocatable,
				Pv_ba_size:        pv.Pv_ba_size,
				Pv_ba_start:       pv.Pv_ba_start,
				Pv_duplicate:      pv.Pv_duplicate,
				Pv_exported:       pv.Pv_exported,
				Pv_ext_vsn:        pv.Pv_ext_vsn,
				Pv_free:           pv.Pv_free,
				Pv_in_use:         pv.Pv_in_use,
				Pv_major:          pv.Pv_major,
				Pv_minor:          pv.Pv_minor,
				Pv_mda_count:      pv.Pv_mda_count,
				Pv_mda_free:       pv.Pv_mda_free,
				Pv_mda_size:       pv.Pv_mda_size,
				Pv_mda_used_count: pv.Pv_mda_used_count,
				Pv_missing:        pv.Pv_missing,
				Pv_pe_alloc_count: pv.Pv_pe_alloc_count,
				Pv_pe_count:       pv.Pv_pe_count,
				Pv_size:           pv.Pv_size,
				Pv_used:           pv.Pv_used,
			}
			InfoList = append(InfoList, Info)
		}
	}
	return InfoList, nil
}

func GetVgInfo(report LvmReportInfo) ([]VgInfo, error) {
	var InfoList []VgInfo = nil

	for _, section := range report.Report {
		for _, vg := range section.VG {
			Info := VgInfo{
				Vg_uuid:             vg.Vg_uuid,
				Lv_count:            vg.Lv_count,
				Max_lv:              vg.Max_lv,
				Max_pv:              vg.Max_pv,
				Pv_count:            vg.Pv_count,
				Snap_count:          vg.Snap_count,
				Vg_clustered:        vg.Vg_clustered,
				Vg_exported:         vg.Vg_exported,
				Vg_extendable:       vg.Vg_extendable,
				Vg_extent_count:     vg.Vg_extent_count,
				Vg_extent_size:      vg.Vg_extent_size,
				Vg_free:             vg.Vg_free,
				Vg_free_count:       vg.Vg_free_count,
				Vg_mda_copies:       vg.Vg_mda_copies,
				Vg_mda_count:        vg.Vg_mda_count,
				Vg_mda_free:         vg.Vg_mda_free,
				Vg_mda_size:         vg.Vg_mda_size,
				Vg_mda_used_count:   vg.Vg_mda_used_count,
				Vg_missing_pv_count: vg.Vg_missing_pv_count,
				Vg_partial:          vg.Vg_partial,
				Vg_seqno:            vg.Vg_seqno,
				Vg_shared:           vg.Vg_shared,
				Vg_size:             vg.Vg_size,
			}
			InfoList = append(InfoList, Info)
		}
	}
	return InfoList, nil
}

func GetLvInfo(report LvmReportInfo) ([]LvInfo, error) {
	var InfoList []LvInfo = nil

	for _, section := range report.Report {
		for _, lv := range section.LV {
			Info := LvInfo{
				Lv_uuid:                     lv.Lv_uuid,
				Cache_dirty_blocks:          lv.Cache_dirty_blocks,
				Cache_read_hits:             lv.Cache_read_hits,
				Cache_read_misses:           lv.Cache_read_misses,
				Cache_total_blocks:          lv.Cache_total_blocks,
				Cache_used_blocks:           lv.Cache_used_blocks,
				Cache_write_hits:            lv.Cache_write_hits,
				Cache_write_misses:          lv.Cache_write_misses,
				Copy_percent:                lv.Copy_percent,
				Data_percent:                lv.Data_percent,
				Integritymismatches:         lv.Integritymismatches,
				Lv_active_exclusively:       lv.Lv_active_exclusively,
				Lv_active_locally:           lv.Lv_active_locally,
				Lv_active_remotely:          lv.Lv_active_remotely,
				Lv_allocation_locked:        lv.Lv_allocation_locked,
				Lv_check_needed:             lv.Lv_check_needed,
				Lv_converting:               lv.Lv_converting,
				Lv_device_open:              lv.Lv_device_open,
				Lv_fixed_minor:              lv.Lv_fixed_minor,
				Lv_historical:               lv.Lv_historical,
				Lv_image_synced:             lv.Lv_image_synced,
				Lv_inactive_table:           lv.Lv_inactive_table,
				Lv_initial_image_sync:       lv.Lv_initial_image_sync,
				Lv_kernel_major:             lv.Lv_kernel_major,
				Lv_kernel_minor:             lv.Lv_kernel_minor,
				Lv_live_table:               lv.Lv_live_table,
				Lv_major:                    lv.Lv_major,
				Lv_merge_failed:             lv.Lv_merge_failed,
				Lv_merging:                  lv.Lv_merging,
				Lv_metadata_size:            lv.Lv_metadata_size,
				Lv_minor:                    lv.Lv_minor,
				Lv_read_ahead:               lv.Lv_read_ahead,
				Lv_size:                     lv.Lv_size,
				Lv_skip_activation:          lv.Lv_skip_activation,
				Lv_snapshot_invalid:         lv.Lv_snapshot_invalid,
				Lv_suspended:                lv.Lv_suspended,
				Lv_time:                     lv.Lv_time,
				Lv_time_removed:             lv.Lv_time_removed,
				Metadata_percent:            lv.Metadata_percent,
				Origin_size:                 lv.Origin_size,
				Raid_max_recovery_rate:      lv.Raid_max_recovery_rate,
				Raid_min_recovery_rate:      lv.Raid_min_recovery_rate,
				Raid_mismatch_count:         lv.Raid_mismatch_count,
				Raid_write_behind:           lv.Raid_write_behind,
				Raidintegrityblocksize:      lv.Raidintegrityblocksize,
				Seg_count:                   lv.Seg_count,
				Snap_percent:                lv.Snap_percent,
				Sync_percent:                lv.Sync_percent,
				Vdo_saving_percent:          lv.Vdo_saving_percent,
				Vdo_used_size:               lv.Vdo_used_size,
				Writecache_error:            lv.Writecache_error,
				Writecache_free_blocks:      lv.Writecache_free_blocks,
				Writecache_total_blocks:     lv.Writecache_total_blocks,
				Writecache_writeback_blocks: lv.Writecache_writeback_blocks,
			}
			InfoList = append(InfoList, Info)
		}
	}
	return InfoList, nil
}

func GetPvSegInfo(report LvmReportInfo) ([]PvSegInfo, error) {
	var InfoList []PvSegInfo = nil

	for _, section := range report.Report {
		for _, pvseg := range section.PVSEG {
			Info := PvSegInfo{
				Pv_uuid:     pvseg.Pv_uuid,
				Lv_uuid:     pvseg.Lv_uuid,
				Pvseg_size:  pvseg.Pvseg_size,
				Pvseg_start: pvseg.Pvseg_start,
			}
			InfoList = append(InfoList, Info)
		}
	}
	return InfoList, nil
}
