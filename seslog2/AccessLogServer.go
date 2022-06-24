package seslog2

import (
	"encoding/json"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"log"
	"strings"
)

type AccessLogServer struct {
	Options
	*ClickHouse

	*syslog.Server
	*syslog.ChannelHandler
	syslog.LogPartsChannel
}

func NewAccessLogServer(options Options) (*AccessLogServer, error) {
	logPartsChannel := make(syslog.LogPartsChannel)
	clickHouse, err := NewClickHouse(options)
	if err != nil {
		return nil, err
	}

	s := &AccessLogServer{
		Options:         options,
		ClickHouse:      clickHouse,
		Server:          syslog.NewServer(),
		ChannelHandler:  syslog.NewChannelHandler(logPartsChannel),
		LogPartsChannel: logPartsChannel,
	}

	s.SetFormat(syslog.RFC3164)
	s.SetHandler(s.ChannelHandler)

	return s, nil
}

func (s *AccessLogServer) Run() error {
	if err := s.ListenUDP(s.Listen); err != nil {
		return err
	}
	log.Printf("Seslog server listen UDP [%s]\n", s.Listen)
	if err := s.Boot(); err != nil {
		return err
	}

	go s.handleLogParts()
	go s.startWatcher()

	s.Wait()

	return nil
}

func (this *AccessLogServer) getLogPart(logParts format.LogParts, field string) (string, bool) {
	logPart, ok := logParts[field]
	if !ok {
		return "", ok
	}
	s, ok := logPart.(string)
	if !ok {
		return "", ok
	}
	return s, ok
}

func notOk(err error) bool {
	if err != nil {
		log.Println(err)
		return true
	}
	return false
}

func (this *AccessLogServer) handleLogParts() {
	for logParts := range this.LogPartsChannel {
		tag, ok := this.getLogPart(logParts, "tag")
		if !ok {
			continue
		}
		content, ok := this.getLogPart(logParts, "content")
		if !ok {
			continue
		}
		content = content[len("escape=json"):]
		if strings.Contains(content, "escape=json") {
			content = content[len("escape=json"):]
		}
		event := make(Event)
		if err := json.Unmarshal([]byte(content), &event); notOk(err) {
			continue
		}
		if err := this.addEvent(tag, event); notOk(err) {
			continue
		}
	}
}
