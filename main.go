package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/iGiant/go-libs/slkclient"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	url = "http://mega-film.top/browse/0/4/0/0?category=4&s_ad=0"
	filmsFile = "names.txt"
	alreadyFound = "found.txt"
	tag = "#index > table > tbody > tr > td:nth-child(2) > a:nth-child(3)"
)

func getBody(url string) io.ReadCloser {
	client := http.Client{}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("код ошибки:", resp.StatusCode)
	}
	return resp.Body
}

func parseString(s string) []string {
	if !strings.Contains(s, "\"") {
		return strings.Fields(s)
	}
	if strings.Count(s, "\"") % 2 != 0 {
		return []string{}
	}
	result := make([]string, 0)
	for i, word := range strings.Split(s, "\"") {
		if i % 2 == 0 {
			if word != "" {
				result = append(result, strings.Fields(word)...)
			}
		} else {
			if word != "" {
				result = append(result, strings.TrimSpace(word))
			}
		}
	}
	return result
}

func getFilesList(fileName string) [][]string {
	body, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}
	result := make([][]string, 0)
	for _, file := range strings.Split(string(body), "\n") {
		if file != "" {
			result = append(result, parseString(strings.Trim(file, "\r")))
		}
	}
	return result
}

func getSerialsFromSite(body io.ReadCloser, tag string) []string {
	document, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal(err)
	}
	result := make([]string, 0)
	document.Find(tag).Each(func(i int, selection *goquery.Selection) {
		name := selection.Text()
		result = append(result, strings.TrimSpace(name))
	})
	return result
}

func all(s string, sub []string) bool {
	for _, item := range sub {
		if strings.Contains(item, "|") {
			if !or(s, strings.Split(item, "|")) {
				return false
			}
		} else if !strings.Contains(strings.ToLower(s), strings.ToLower(item)) {
			return false
		}
	}
	return true
}

func or(s string, sub []string) bool {
	for _, item := range sub {
		if strings.Contains(strings.ToLower(s), strings.ToLower(item)) {
			return true
		}
	}
	return false
}

func any(s string, sub []string) bool {
	for _, item := range sub {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func findContains(serials []string, names [][]string) []string {
	result := make([]string, 0)
	for _, serial := range serials {
		for _, name := range names {
			if all(serial, name) {
				result = append(result, serial)
			}
		}
	}
	return result
}

func filterAlreadyFound(fileName string, serials []string) []string {
	body, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal(err)
	}
	already := make([]string, 0)
	for _, file := range strings.Split(string(body), "\n") {
		if file != "" {
			already = append(already, strings.Trim(file, "\r"))
		}
	}
	result := make([]string, 0)
	for _, serial := range serials {
		if !any(serial, already) {
			result = append(result, serial)
		}
	}
	if len(result) != 0 {
		already = append(already, result...)
		_ = ioutil.WriteFile(fileName, []byte(strings.Join(already, "\n")), 0644)
	}
	return result
}


func main() {
	body := getBody(url)
	files := getFilesList(filmsFile)
	serials := getSerialsFromSite(body, tag)
	result := filterAlreadyFound(alreadyFound, findContains(serials, files))
	if len(result) > 0 {
		var text string
		if len(result) == 1 {
			text = "Появился сериал: " + result[0]
		} else {
			text = "Появились сериалы:\n" + strings.Join(result, "\n")
		}
		_ = slkclient.SendToSlack(":film_frames: Serial", text, "@sergey_gr", "", "")
	}
}
