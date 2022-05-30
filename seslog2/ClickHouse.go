package seslog2

import (
	"context"
	"errors"
	"fmt"
	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"time"
)

type ClickHouse struct {
	Options
	driver.Conn
	ctx context.Context
	//typedEvents []interface{}
	eventQueues EventQueueByTag
}

func NewClickHouse(options Options) (*ClickHouse, error) {
	addr := fmt.Sprintf("%s:%d", options.Clickhouse.Host, options.Clickhouse.Port)
	conn, err := ch.Open(&ch.Options{
		Addr: []string{addr},
		Auth: ch.Auth{
			Database: options.Clickhouse.Database,
			Username: options.Clickhouse.User,
			Password: options.Clickhouse.Password,
		},
		//Debug:           true,
		DialTimeout:     time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		return nil, err
	}

	c := &ClickHouse{
		Options:     options,
		Conn:        conn,
		ctx:         context.Background(),
		eventQueues: EventQueueByTag{},
	}
	return c, nil
}

func (clickHouse *ClickHouse) startWatcher() {
	dur, err := time.ParseDuration(clickHouse.Options.FlushInterval)
	if err != nil {
		dur = 50 * time.Second
	}
	for range time.Tick(dur) {
		clickHouse.writeEvents()
	}
}

func (clickHouse *ClickHouse) getEventQueue(tag string) *EventQueue {
	q, ok := clickHouse.eventQueues[tag]
	if !ok {
		q = NewEventQueue(tag, clickHouse)
		clickHouse.eventQueues[tag] = q
	}
	return q
}

func (clickHouse *ClickHouse) addEvent(tag string, event Event) error {
	if tag == "" {
		return errors.New("empty tag")
	}
	q := clickHouse.getEventQueue(tag)
	return q.addEvent(event)
}

func (clickHouse *ClickHouse) writeEvents() {
	for _, q := range clickHouse.eventQueues {
		q.writeEvents()
		//err := q.writeEvents(clickHouse)
		//if err != nil {
		//	log.Println(err)
		//}
	}
}
