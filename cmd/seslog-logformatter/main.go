package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: seslog-logformat-compiler <log_format_file.json>")
		return
	}
	name := strings.ReplaceAll(filepath.Base(os.Args[1]), ".json", "")
	b, err := ioutil.ReadFile(os.Args[1])
	check(err)
	log_format := make(map[string]interface{})
	err = json.Unmarshal(b, &log_format)
	check(err)

	keys := make([]string, len(log_format))
	{
		i := 0
		for k := range log_format {
			keys[i] = k
			i++
		}
	}
	sort.Strings(keys)

	fmt.Printf("log_format %s_log_format escape=json\n'{'\n", name)
	first := true
	for _, k := range keys {
		v := log_format[k]

		if first {
			first = false
			fmt.Printf("\t'")
		} else {
			fmt.Printf(",'\n\t'")
		}
		fmt.Printf("\"%s\": \"%s\"", k, v)
	}
	fmt.Printf("'\n'}';")
}
