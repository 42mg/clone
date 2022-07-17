package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/theckman/yacspin"
	"github.com/tidwall/gjson"
)

func main() {
	silent := flag.Bool("s", false, "silent")
	flag.Parse()
	token, a := os.Getenv("GitHubToken"), flag.Args()
	if token == "" {
		log.Fatalln("missing required environment variable.")
	}
	if len(a) == 0 {
		log.Fatalln("missing required argument.")
	} else if len(a) > 1 {
		log.Fatalln("too many arguments.")
	}
	user := a[0]
	var clone_urls []string
	i := 1
	for {
		req, err := http.NewRequest("GET", "https://api.github.com/users/"+user+"/repos?per_page=100&page="+strconv.Itoa(i), nil)
		if err != nil {
			log.Fatalln(err)
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("Authorization", "token "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer resp.Body.Close()
		respData, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		jsonData := string(respData)
		if m := gjson.Get(jsonData, "message"); m.Exists() {
			log.Fatalln(m)
		}
		if gjson.Get(jsonData, "#").Int() == 0 {
			break
		}
		repos := gjson.GetMany(jsonData, "#.fork", "#.clone_url")
		for i, v := range repos[0].Array() {
			if !v.Bool() {
				clone_urls = append(clone_urls, repos[1].Array()[i].String())
			}
		}
		i++
	}
	if len(clone_urls) > 0 {
		err := os.Mkdir(user, 0755)
		if err != nil {
			log.Fatalln(err)
		}
	}
	cfg := yacspin.Config{
		Frequency:     100 * time.Millisecond,
		CharSet:       yacspin.CharSets[59],
		Message:       " cloning into user " + strconv.Quote(user),
		StopCharacter: "✓",
		StopColors:    []string{"fgGreen"},
		StopMessage:   " cloned " + strconv.Itoa(len(clone_urls)) + " repositories.",
	}
	spinner, err := yacspin.New(cfg)
	if err != nil {
		log.Fatalln(err)
	}
	if !*silent {
		err = spinner.Start()
		if err != nil {
			log.Fatalln(err)
		}
	}
	var wg sync.WaitGroup
	for _, v := range clone_urls {
		v := v
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command("git", "-C", user, "clone", "-q", v)
			err := cmd.Run()
			if err != nil {
				log.Fatalln(err)
			}
		}()
	}
	wg.Wait()
	if !*silent {
		err = spinner.Stop()
		if err != nil {
			log.Fatalln(err)
		}
	}
}
