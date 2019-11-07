package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/go-ini/ini"
	"github.com/iGiant/go-slack_client"
	"github.com/iGiant/proxies"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

const (
	filename = "films.ini"
)

type site struct {
	url          string
	subUrl       string
	filmsFile    string
	alreadyFound string
	tag          string
}

func getFromIni(fileName string) (site, error) {
	cfg, err := ini.Load(fileName)
	if err != nil {
		return site{}, err
	}
	result := site{
		url:          cfg.Section("site").Key("url").String(),
		subUrl:       cfg.Section("site").Key("subUrl").String(),
		filmsFile:    cfg.Section("site").Key("filmsFile").String(),
		alreadyFound: cfg.Section("site").Key("alreadyFound").String(),
		tag:          strings.ReplaceAll(cfg.Section("site").Key("tag").String(), "№", "#"),
	}
	return result, nil
}

func parseString(s string) []string {
	if !strings.Contains(s, "\"") {
		return strings.Fields(s)
	}
	if strings.Count(s, "\"")%2 != 0 {
		return []string{}
	}
	result := make([]string, 0)
	for i, word := range strings.Split(s, "\"") {
		if i%2 == 0 {
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

func getFilmsFromSite(body io.ReadCloser, tag string) []string {
	document, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		log.Fatal(err)
	}
	result := make([]string, 0)
	document.Find(tag).Each(func(i int, selection *goquery.Selection) {
		name := selection.Text()
		if isQuality(name) {
			result = append(result, strings.TrimSpace(name))
		}
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

func isQuality(name string) bool {
	temp := strings.Split(name, "|")
	quality := strings.TrimSpace(temp[len(temp)-1])
	return strings.EqualFold(quality, "iTunes") || strings.EqualFold(quality, "Лицензия")
}

func filterAlreadyFound(fileName string, serials []string) []string {
	body, _ := ioutil.ReadFile(fileName)
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
		_ = ioutil.WriteFile(fileName, []byte(strings.Join(already, "\n")), 0777)
	}
	return result
}

func main() {
	param, err := getFromIni(filename)
	if err != nil {
		os.Exit(1)
	}
	var proxiesList []string
	for i := 0; i < 3; i++ {
		proxiesList, err = proxies.GetProxiesList()
		if err == nil {
			break
		}
	}
	if err != nil {
		_ = slack_client.SendToSlack(":film_frames: Films (Ошибка)",
			"Недоступен сайт с proxy",
			"@sergey_gr",
			"",
			"",
		)
		os.Exit(2)
	}
	needFilms := getFilmsFromTrello()
	if len(needFilms) == 0 {
		needFilms = getFilesList(param.filmsFile)
		if len(needFilms) == 0 {
			os.Exit(3)
		}
	}
	result := make([]string, 0)
	for _, film := range needFilms {
		u, _ := url.Parse(param.url)
		u.Path = path.Join(u.Path, param.subUrl, film[0])
		var response *http.Response
		for _, proxy := range proxiesList {
			response, err = proxies.GetSite(u.String(), proxy)
			if err == nil {
				break
			}
		}
		if err != nil {
			_ = slack_client.SendToSlack(
				":film_frames: Films (Ошибка)",
				"Нет рабочих proxy-серверов",
				"@sergey_gr",
				"",
				"",
			)
			os.Exit(4)
		}
		films := getFilmsFromSite(response.Body, param.tag)
		_ = response.Body.Close()
		result = append(result, filterAlreadyFound(param.alreadyFound, findContains(films, needFilms))...)
	}
	if len(result) > 0 {
		var text string
		if len(result) == 1 {
			text = "Появился фильм: " + result[0]
		} else {
			text = "Появились фильмы:\n" + strings.Join(result, "\n")
		}
		_ = slack_client.SendToSlack(":film_frames: Films", text, "@sergey_gr", "", "")
	}
}
