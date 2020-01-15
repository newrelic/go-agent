package main

// This is a utility designed to create a list of all Elasticsearch http
// endpoints.  The output of this script is checked in as 'endpoints.txt'.

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type endpoint struct {
	firstUnderscoreSegment string
	filename               string
	method                 string
	path                   string
}

func main() {
	if len(os.Args) < 2 {
		panic("provide path to github.com/elastic/elasticsearch/tree/7.5/rest-api-spec/src/main/resources/rest-api-spec/api/")
	}
	var files []string
	err := filepath.Walk(os.Args[1], func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}

	var endpoints []endpoint

	for _, path := range files {
		_, filename := filepath.Split(path)
		// Check the "_" prefix to avoid parsing _common.
		if filename == "" || strings.HasPrefix(filename, "_") {
			continue
		}

		contents, err := ioutil.ReadFile(path)
		if nil != err {
			panic(err)
		}
		var fields map[string]struct {
			Stability string `json:"stability"`
			URL       struct {
				Paths []struct {
					Path    string   `json:"path"`
					Methods []string `json:"methods"`
				} `json:"paths"`
			} `json:"url"`
		}
		err = json.Unmarshal(contents, &fields)
		if nil != err {
			panic(err)
		}
		for _, v := range fields {
			for _, p := range v.URL.Paths {
				path := p.Path
				segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
				var firstUnderscoreSegment string
				for _, s := range segments {
					if strings.HasPrefix(s, "_") {
						firstUnderscoreSegment = s
						break
					}
				}

				for _, method := range p.Methods {
					endpoints = append(endpoints, endpoint{
						firstUnderscoreSegment: firstUnderscoreSegment,
						filename:               filename,
						method:                 method,
						path:                   path,
					})
				}
			}
		}
	}

	sort.Slice(endpoints, func(i int, j int) bool {
		iseg := endpoints[i].firstUnderscoreSegment
		jseg := endpoints[j].firstUnderscoreSegment
		if iseg == jseg {
			return endpoints[i].filename < endpoints[j].filename
		}
		return iseg < jseg
	})

	for _, e := range endpoints {
		fmt.Printf("%-18v %-35v %-10v %v\n",
			e.firstUnderscoreSegment,
			e.filename,
			e.method,
			e.path)
	}
}
