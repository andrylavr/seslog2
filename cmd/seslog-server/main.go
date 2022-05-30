package main

import (
	"github.com/andrylavr/seslog2/seslog2"
	"os"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func findConfigFile() (string, error) {
	files := []string{
		"seslog.json",
		"/etc/seslog2/seslog.json",
	}
	for _, file := range files {
		if _, err := os.Stat(file); err == nil {
			return file, nil
		}
	}
	return "", os.ErrNotExist
}

func main() {
	configFile, err := findConfigFile()
	check(err)
	b, err := os.ReadFile(configFile)
	check(err)
	options, err := seslog2.ParseOptions(b)
	check(err)
	server, err := seslog2.NewAccessLogServer(options)
	check(err)
	err = server.Run()
	check(err)
}
