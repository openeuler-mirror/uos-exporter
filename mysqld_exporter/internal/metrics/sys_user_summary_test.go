package metrics

//func TestSysUserSummary(t *testing.T) {
//	db, mock, err := sqlmock.New()
//	if err != nil {
//		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
//	}
//	defer db.Close()
//	columns := []string{
//		"user",
//		"statements",
//		"statement_latency",
//		"table_scans",
//		"file_ios",
//		"file_io_latency",
//		"current_connections",
//		"total_connections",
//		"unique_hosts",
//		"current_memory",
//		"total_memory_allocated",
//	}
//	rows := sqlmock.NewRows(columns).
//		AddRow("root",
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1).
//		AddRow("mysql.sys",
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1).
//		AddRow("",
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1,
//			1)
//	mock.ExpectQuery(regexp.QuoteMeta(sysUserSummaryQuery)).WillReturnRows(rows)
//	qd := NewSysUserSummary(mysql.Instance{Db: db})
//	ch := make(chan prometheus.Metric, 100)
//	qd.Collect(ch)
//	close(ch)
//	metricCount := 0
//	for range ch {
//		metricCount++
//	}
//	if metricCount == 0 {
//		t.Error("no metrics were collected")
//	}
//	if err := mock.ExpectationsWereMet(); err != nil {
//		t.Errorf("there were unfulfilled expectations: %s", err)
//	}
//}
// Part 2 commit for mysqld_exporter/internal/metrics/sys_user_summary_test.go
