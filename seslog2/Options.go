package seslog2

import (
	"encoding/json"
	"time"
)

type Options struct {
	Clickhouse struct {
		User     string `json:"user"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Database string `json:"database"`
	} `json:"clickhouse"`
	Listen        string `json:"listen"`
	FlushInterval string `json:"flush_interval"`
	Retry         int    `json:"retry"`
	RetryTimeout  string `json:"retry_timeout"`
	retryTimeout  time.Duration
	WriteOnFail   string `json:"write_on_fail"`
}

func ParseOptions(b []byte) (Options, error) {
	options := Options{}
	err := json.Unmarshal(b, &options)
	if err != nil {
		return options, err
	}

	retryTimeout, err := time.ParseDuration(options.RetryTimeout)
	if err != nil {
		retryTimeout = time.Second * 30
	}
	options.retryTimeout = retryTimeout

	return options, nil
}
