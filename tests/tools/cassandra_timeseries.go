package tools

import (
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/uol/hashing"
)

type cassTs struct {
	cql *gocql.Session
}

func (ts *cassTs) init(cql *gocql.Session) {
	ts.cql = cql
}

type TableProperties struct {
	Bloom_filter_fp_chance      float64
	Caching                     map[string]string
	Comment                     string
	Compaction                  map[string]string
	Compression                 map[string]string
	Dclocal_read_repair_chance  float64
	Default_time_to_live        int
	Gc_grace_seconds            int
	Max_index_interval          int
	Memtable_flush_period_in_ms int
	Min_index_interval          int
	Read_repair_chance          float64
	Speculative_retry           string
}

type KeyspaceAttributes struct {
	Name               string
	Replication_factor int
	Datacenter         string
	Contact            string
}

type KeyspaceProperties struct {
	Keyspace_name  string
	Durable_writes bool
	Replication    map[string]string
}

const (
	cqlKeyspaceTables     = `SELECT table_name FROM system_schema.tables WHERE keyspace_name = ?`
	cqlExists             = `SELECT keyspace_name FROM system_schema.keyspaces WHERE keyspace_name = ?`
	cqlExistsInformation  = `SELECT key, datacenter, contact, replication_factor FROM mycenae.ts_keyspace WHERE key = ?`
	cqlTableProperties    = `SELECT bloom_filter_fp_chance, caching, comment, compaction, compression, dclocal_read_repair_chance, default_time_to_live, gc_grace_seconds, max_index_interval, memtable_flush_period_in_ms, min_index_interval, read_repair_chance, speculative_retry from system_schema.tables  where keyspace_name = ? and table_name = ?`
	cqlKeyspaceProperties = `SELECT keyspace_name, durable_writes, replication from system_schema.keyspaces where keyspace_name = ?`
	cqlDropKS             = `DROP KEYSPACE %v`
	cqlSelectKS           = `SELECT key, contact, datacenter, replication_factor FROM mycenae.ts_keyspace WHERE key = ?`
	cqlDeleteKS           = `DELETE FROM mycenae.ts_keyspace WHERE key = '%v'`
	cqlInsertKS           = `INSERT INTO mycenae.ts_keyspace (key, datacenter, contact, replication_factor) VALUES ('%v', 'dc_gt_a1', 'test@test.com', 1)`
)

var (
	TTLKeyspaceMap = map[int]string{
		1: "one_day",
		3: "three_days",
		7: "one_week",
	}
)

func (ts *cassTs) getTTLKeyspace(ttl int) string {
	if ks, ok := TTLKeyspaceMap[ttl]; !ok {
		panic("no ttl keyspace with value " + strconv.FormatInt(int64(ttl), 10))
	} else {
		return ks
	}
}

func (ts *cassTs) CountValuesPriorDate(ttl int, id string, dateBegin int64) int {
	it := ts.cql.Query(fmt.Sprintf("SELECT id FROM %s.ts_number_stamp WHERE id=? AND date >= ?", ts.getTTLKeyspace(ttl)), id, time.Unix(dateBegin, 0).UTC()).Iter()

	var count int
	var scanned string
	for it.Scan(&scanned) {
		count++
	}
	if err := it.Close(); err != nil {
		log.Println(err)
	}
	return count
}

func (ts *cassTs) CountTextPriorDate(ttl int, id string, dateBegin int64) int {
	it := ts.cql.Query(fmt.Sprintf("SELECT id FROM %s.ts_text_stamp WHERE id=? AND date >= ?", ts.getTTLKeyspace(ttl)), id, time.Unix(dateBegin, 0).UTC()).Iter()

	var count int
	var scanned string
	for it.Scan(&scanned) {
		count++
	}
	if err := it.Close(); err != nil {
		log.Println(err)
	}
	return count
}

func (ts *cassTs) GetTextFromTwoDatesSTAMP(ttl int, id string, dateBeforeRequest time.Time, dateAfterRequest time.Time) (nValue string) {
	if err := ts.cql.Query(fmt.Sprintf(`SELECT value FROM %s.ts_text_stamp WHERE id=? AND date >= ? AND date <= ?`, ts.getTTLKeyspace(ttl)),
		id,
		dateBeforeRequest.Truncate(time.Second),
		dateAfterRequest.Truncate(time.Second),
	).Scan(&nValue); err != nil && err != gocql.ErrNotFound {
		log.Println(err)
	}
	return
}

func GetHashFromMetricAndTags(metric string, tags map[string]string) string {

	numParameters := (len(tags) * 2) + 1
	strParameters := make([]string, numParameters)
	strParameters[0] = metric

	i := 1
	for k, v := range tags {
		strParameters[i] = k
		i++
		strParameters[i] = v
		i++
	}

	sort.Strings(strParameters)

	parameters := make([]interface{}, numParameters)
	for i, v := range strParameters {
		parameters[i] = v
	}

	hash, err := hashing.GenerateSHAKE128(16, parameters...)
	if err != nil {
		panic(err)
	}

	return hex.EncodeToString(hash)
}

func GetTextHashFromMetricAndTags(metric string, tags map[string]string) string {

	return fmt.Sprint("T", GetHashFromMetricAndTags(metric, tags))
}

func (ts *cassTs) CountKeyspaces() int {
	it := ts.cql.Query("SELECT id FROM system_schema.keyspaces").Iter()

	var count int
	var scanned string
	for it.Scan(&scanned) {
		count++
	}
	if err := it.Close(); err != nil {
		log.Println(err)
	}
	return count
}

//func (ts *cassTs) CountKeyspaces() (count int) {
//	if err := ts.cql.Query(cqlCountKeyspaces).Scan(&count); err != nil {
//		log.Println(err)
//	}
//	return
//}

func (ts *cassTs) CountTsKeyspaces() int {
	it := ts.cql.Query("SELECT id FROM mycenae.ts_keyspace").Iter()

	var count int
	var scanned string
	for it.Scan(&scanned) {
		count++
	}
	if err := it.Close(); err != nil {
		log.Println(err)
	}
	return count
}

//func (ts *cassTs) CountTsKeyspaces() (count int) {
//	if err := ts.cql.Query(cqlCountTsKeyspaces).Scan(&count); err != nil {
//		log.Println(err)
//	}
//	return
//}

func (ts *cassTs) CountTsKeyspaceByKsid(keyspace string) int {
	it := ts.cql.Query("SELECT key FROM mycenae.ts_keyspace WHERE key=?", keyspace).Iter()

	var count int
	var scanned string
	for it.Scan(&scanned) {
		if keyspace == scanned {
			count++
		}
	}
	if err := it.Close(); err != nil {
		log.Println(err)
	}
	return count
}

//func (ts *cassTs) CountTsKeyspaceByName(name string) int {
//	var count1, count2 int
//	if err := ts.cql.Query(`SELECT count(*) FROM mycenae.ts_keyspace WHERE name = ? AND token(key) < 0;`, name).Scan(&count1); err != nil {
//		log.Println(err)
//	}
//	if err := ts.cql.Query(`SELECT count(*) FROM mycenae.ts_keyspace WHERE name = ? AND token(key) >= 0;`, name).Scan(&count2); err != nil {
//		log.Println(err)
//	}
//	return count1 + count2
//}

func (ts *cassTs) CountKeyspacesNoCassWarning() int {
	var count1, count2 int
	if err := ts.cql.Query(`SELECT count(*) FROM system_schema.keyspaces WHERE token(keyspace_name) < 0;`).Scan(&count1); err != nil {
		log.Println(err)
	}
	if err := ts.cql.Query(`SELECT count(*) FROM system_schema.keyspaces WHERE token(keyspace_name) >= 0;`).Scan(&count2); err != nil {
		log.Println(err)
	}
	return count1 + count2
}

func (ts *cassTs) CountTsKeyspacesNoCassWarning() int {
	var count1, count2 int
	if err := ts.cql.Query(`SELECT count(*) FROM mycenae.ts_keyspace WHERE token(key) < 0;`).Scan(&count1); err != nil {
		log.Println(err)
	}
	if err := ts.cql.Query(`SELECT count(*) FROM mycenae.ts_keyspace WHERE token(key) >= 0;`).Scan(&count2); err != nil {
		log.Println(err)
	}
	return count1 + count2
}

func (ts *cassTs) KeyspaceTables(keyspace string) (tables []string) {
	iter := ts.cql.Query(
		cqlKeyspaceTables,
		keyspace,
	).Iter()

	var table string

	for iter.Scan(&table) {
		tables = append(tables, table)
	}
	if err := iter.Close(); err != nil {
		log.Println(err)
	}
	return
}

func (ts *cassTs) Exists(keyspace string) bool {
	it := ts.cql.Query(cqlExists, keyspace).Iter()
	var err error
	var count int
	var scanned string
	for it.Scan(&scanned) {
		if keyspace == scanned {
			count++
		}
	}
	if err = it.Close(); err != nil {
		log.Println(err)
	}
	return err == nil && count == 1
}

func (ts *cassTs) ExistsInformation(keyspace string, replication_factor int, datacenter string, contact string) bool {
	var err error
	var count, scRepFactor int
	var scKey, scDatacenter, scContact string
	it := ts.cql.Query(
		cqlExistsInformation,
		keyspace,
	).Iter()
	for it.Scan(&scKey, &scDatacenter, &scContact, &scRepFactor) {
		if scKey == keyspace && scDatacenter == datacenter && scContact == contact &&
			scRepFactor == replication_factor {
			count++
		}
	}
	if err = it.Close(); err != nil {
		log.Println(err)
	}
	return err == nil && count == 1

}

func (ts *cassTs) TableProperties(keyspace string, table string) TableProperties {
	var caching, compaction, compression map[string]string
	var speculative_retry, comment string
	var default_time_to_live, gc_grace_seconds, max_index_interval, memtable_flush_period_in_ms,
		min_index_interval int
	var bloom_filter_fp_chance, dclocal_read_repair_chance, read_repair_chance float64

	if err := ts.cql.Query(cqlTableProperties,
		keyspace,
		table,
	).Scan(&bloom_filter_fp_chance, &caching, &comment, &compaction, &compression, &dclocal_read_repair_chance,
		&default_time_to_live, &gc_grace_seconds, &max_index_interval, &memtable_flush_period_in_ms, &min_index_interval,
		&read_repair_chance, &speculative_retry); err != nil {
		log.Println(err)
	}

	return TableProperties{
		Bloom_filter_fp_chance:      bloom_filter_fp_chance,
		Caching:                     caching,
		Comment:                     comment,
		Compaction:                  compaction,
		Compression:                 compression,
		Dclocal_read_repair_chance:  dclocal_read_repair_chance,
		Default_time_to_live:        default_time_to_live,
		Gc_grace_seconds:            gc_grace_seconds,
		Max_index_interval:          max_index_interval,
		Memtable_flush_period_in_ms: memtable_flush_period_in_ms,
		Min_index_interval:          min_index_interval,
		Read_repair_chance:          read_repair_chance,
		Speculative_retry:           speculative_retry,
	}
}

func (ts *cassTs) Drop(keyspace string) bool {

	err := ts.cql.Query(
		fmt.Sprintf(cqlDropKS, keyspace),
	).Exec()

	return err == nil

}

func (ts *cassTs) Delete(keyspace string) bool {

	err := ts.cql.Query(
		fmt.Sprintf(cqlDeleteKS, keyspace),
	).Exec()

	return err == nil

}

func (ts *cassTs) Insert(keyspace string) bool {

	err := ts.cql.Query(
		fmt.Sprintf(cqlInsertKS, keyspace),
	).Exec()

	return err == nil

}

func (ts *cassTs) KsAttributes(keyspace string) KeyspaceAttributes {
	var key, datacenter, contact string
	var replication_factor int

	if err := ts.cql.Query(cqlSelectKS,
		keyspace,
	).Scan(&key, &contact, &datacenter, &replication_factor); err != nil {
		log.Println(err)
	}
	return KeyspaceAttributes{
		Name:               key,
		Datacenter:         datacenter,
		Contact:            contact,
		Replication_factor: replication_factor,
	}
}

func (ts *cassTs) KeyspaceProperties(keyspace string) KeyspaceProperties {
	var keyspace_name string
	var durable_writes bool
	var replication map[string]string

	if err := ts.cql.Query(cqlKeyspaceProperties,
		keyspace,
	).Scan(&keyspace_name, &durable_writes, &replication); err != nil {
		log.Println(err)
	}
	return KeyspaceProperties{
		Keyspace_name:  keyspace_name,
		Durable_writes: durable_writes,
		Replication:    replication,
	}
}
