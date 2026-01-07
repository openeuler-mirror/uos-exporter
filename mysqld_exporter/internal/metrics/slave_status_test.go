package metrics

//func TestSlaveStatus(t *testing.T) {
//	db, mock, err := sqlmock.New()
//	if err != nil {
//		t.Fatalf("error opening a stub database connection: %s", err)
//	}
//	defer db.Close()
//	columns := []string{
//		"Master_Host",
//		"Read_Master_Log_Pos",
//		"Slave_IO_Running",
//		"Slave_SQL_Running",
//		"Seconds_Behind_Master",
//	}
//	rows := sqlmock.NewRows(columns).
//		AddRow("127.0.0.1", "1", "Connecting", "Yes", "2")
//	mock.ExpectQuery(regexp.QuoteMeta("SHOW SLAVE STATUS")).WillReturnRows(rows)
//	qd := NewSlaveStatus(mysql.Instance{Db: db})
//	ch := make(chan prometheus.Metric, 100)
//	qd.Collect(ch)
//	close(ch)
//	//metricCount := 0
//	//for range ch {
//	//	metricCount++
//	//}
//	//if metricCount == 0 {
//	//	t.Error("no metrics were collected")
//	//}
//	//if err := mock.ExpectationsWereMet(); err != nil {
//	//	t.Errorf("there were unfulfilled expectations: %s", err)
//	//}
//}
// Part 2 commit for mysqld_exporter/internal/metrics/slave_status_test.go
