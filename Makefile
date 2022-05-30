LDFLAGS=-ldflags "-s -w"

builds:
	@[ -d build ] || mkdir -p build
	go build ${LDFLAGS} -o build/seslog-server cmd/seslog-server/main.go
	@file  build/seslog-server
	@du -h build/seslog-server

buildf:
	@[ -d build ] || mkdir -p build
	go build ${LDFLAGS} -o build/seslog-logformatter cmd/seslog-logformatter/main.go
	@file  build/seslog-logformatter
	@du -h build/seslog-logformatter

build: builds buildf

delete:
	rm -f build/seslog-server
	rm -f build/seslog-logformatter

install:
	mkdir -p /opt/seslog2
	cp build/seslog-* /opt/seslog2
	cp seslog.example.json /opt/seslog2/seslog.json
	cp package/systemd/seslog-server.service /etc/systemd/system/seslog2.service
	/bin/systemctl daemon-reload
	/bin/systemctl enable seslog2
	service seslog2 start
	service seslog2 status

d:
	docker-compose -f dockerfiles/docker-compose.yml rm --force
	docker-compose -f dockerfiles/docker-compose.yml up --build