package seslog2

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

type EventQueue struct {
	tableName   string
	typedEvents []TypedEvent
	fields      map[string]VarType
	fieldNames  []string
	engine      string
	ttl         time.Duration
	keepAsArray bool
	*ClickHouse
}

type EventQueueByTag map[string]*EventQueue

var ttlRe, _ = regexp.Compile(`_ttl_(\d+[a-z]+)`)

func NewEventQueue(tag string, clickHouse *ClickHouse) *EventQueue {
	tableName := strings.ReplaceAll(tag, "_null", "")
	tableName = strings.ReplaceAll(tableName, "_array", "")
	tableName = ttlRe.ReplaceAllString(tableName, "")

	q := &EventQueue{
		tableName:   tableName,
		typedEvents: []TypedEvent{},
		fields:      map[string]VarType{},
		fieldNames:  []string{},
		//ttl:         60 * 24 * time.Hour, //60 day
		ttl:        0,
		ClickHouse: clickHouse,
	}
	if strings.Contains(tag, "_null") {
		q.engine = "Null"
	} else {
		q.engine = "MergeTree"
	}
	q.keepAsArray = strings.Contains(tag, "_array")
	m := ttlRe.FindAllStringSubmatch(tag, 1)
	if len(m) > 0 && len(m[0]) > 1 {
		q.ttl, _ = ParseDuration(m[0][1])
	}

	return q
}

func (q *EventQueue) FieldNames() []string {
	fieldNames := make([]string, len(q.fields))
	i := 0
	for k := range q.fields {
		fieldNames[i] = k
		i++
	}
	sort.Strings(fieldNames)
	return fieldNames
}

func (q *EventQueue) addEvent(event Event) error {
	if len(event) == 0 {
		return errors.New("empty event")
	}

	if q.keepAsArray {
		return errors.New("not available now")
	}

	typedEvent := TypedEvent{}
	for k, v := range event {
		var varType VarType
		if _, ok := q.fields[k]; ok {
			varType = q.fields[k]
		} else {
			varType, k = fieldNameToVarType(k)
			q.fields[k] = varType
			q.fieldNames = q.FieldNames()
		}
		typedEvent[k] = varType.convert(v)
	}

	q.typedEvents = append(q.typedEvents, typedEvent)

	return nil
}

func (q *EventQueue) sortedFields() (keys []string, vals []VarType) {
	keys = make([]string, len(q.fields))
	vals = make([]VarType, len(q.fields))
	{
		i := 0
		for k := range q.fields {
			keys[i] = k
			i++
		}
	}
	sort.Strings(keys)
	for i, k := range keys {
		vals[i] = q.fields[k]
	}
	return
}

func (q *EventQueue) createColumns() error {
	fKeys, fVals := q.sortedFields()
	for i, fieldName := range fKeys {
		if fieldName == "time" {
			continue
		}
		fieldType := fVals[i]
		fieldTypeStr, ok := varTypeToClickHouse[fieldType]
		if !ok {
			fieldTypeStr = "String"
		}
		alterSQL := fmt.Sprintf(
			"ALTER TABLE `%s` ADD COLUMN IF NOT EXISTS `%s` %s",
			q.tableName,
			fieldName,
			fieldTypeStr,
		)
		if err := q.Exec(q.ctx, alterSQL); err != nil {
			return err
		}
	}

	return nil
}

func (q *EventQueue) createTable() error {

	var createSQL = fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS `%s` (time DateTime DEFAULT now(), date Date DEFAULT toDate(time), wtime DateTime DEFAULT now()) ",
		q.tableName,
	)

	if q.engine == "Null" {
		createSQL += "ENGINE = Null"
	} else {
		createSQL += fmt.Sprintf(
			"ENGINE = %s() ORDER BY time",
			q.engine,
		)
		ttl := int(math.Round(q.ttl.Seconds()))
		if ttl > 0 {
			createSQL += fmt.Sprintf(
				" TTL time + INTERVAL %d SECOND",
				ttl,
			)
		}
	}

	return q.Exec(q.ctx, createSQL)
}

func (q *EventQueue) Try(retry *int, f func() error, onFail func()) bool {
	err := f()
	if err == nil {
		return true
	}

	for {
		*retry++
		log.Println("Retry")
		err = f()
		if err == nil {
			return true
		}
		log.Println(err)
		if *retry < q.Options.Retry {
			log.Printf("Sleep %s\n", q.Options.retryTimeout)
			time.Sleep(q.Options.retryTimeout)
		} else {
			log.Println("Fail")
			break
		}
	}

	onFail()

	return false
}

func (q *EventQueue) writeOnFail(typedEvents []TypedEvent) {
	writeDir := q.Options.WriteOnFail
	if writeDir == "" {
		return
	}

	//mkdir
	if _, err := os.Stat(writeDir); os.IsNotExist(err) {
		err := os.Mkdir(writeDir, 0644)
		if err != nil {
			log.Println(err)
		}
	}

	filename := fmt.Sprintf("%s-%d.json", q.tableName, time.Now().UnixMilli())
	filename = path.Join(writeDir, filename)
	var data []byte
	newRow := []byte("\n")
	for i, typedEvent := range typedEvents {
		row, err := json.Marshal(typedEvent)
		if err != nil {
			log.Println(err)
			continue
		}
		if i > 0 {
			data = append(data, newRow...)
			data = append(data, row...)
		} else {
			data = row
		}
	}

	log.Println("WriteFile", filename, "size = ", len(data))
	err := ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Println(err)
	}

}

func (q *EventQueue) prepareBatch() (driver.Batch, error) {
	fieldsSQL := ""
	for i, k := range q.fieldNames {
		if i > 0 {
			fieldsSQL += ", "
		}
		fieldsSQL += fmt.Sprintf("`%s`", k)
	}
	insertSQL := fmt.Sprintf("INSERT INTO `%s` (%s)", q.tableName, fieldsSQL)
	return q.PrepareBatch(q.ctx, insertSQL)
}

type Row []interface{}

func (q *EventQueue) writeEvents() {
	if len(q.typedEvents) == 0 {
		return
	}

	typedEvents := make([]TypedEvent, len(q.typedEvents))
	copy(typedEvents, q.typedEvents)
	q.typedEvents = q.typedEvents[0:0]

	writeOnFail := func() {
		q.writeOnFail(typedEvents)
	}

	retry := 0
	ok := false
	ok = q.Try(
		&retry,
		func() error {
			return q.createTable()
		},
		writeOnFail,
	)
	if !ok {
		return
	}
	ok = q.Try(
		&retry,
		func() error {
			return q.createColumns()
		},
		writeOnFail,
	)
	if !ok {
		return
	}
	var batch driver.Batch
	ok = q.Try(
		&retry,
		func() error {
			var err error
			batch, err = q.prepareBatch()
			return err
		},
		writeOnFail,
	)
	if !ok {
		return
	}

	row := make(Row, len(q.fields))
	for _, typedEvent := range typedEvents {
		row = row[0:0]
		for _, fName := range q.fieldNames {
			row = append(row, typedEvent[fName])
		}
		err := batch.Append(row...)
		if err != nil {
			log.Println(err)
		}
	}

	q.Try(
		&retry,
		func() error {
			return batch.Send()
		},
		writeOnFail,
	)
}
