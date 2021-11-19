package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/probakowski/go-viessmann"
	"html/template"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

//go:embed static
var web embed.FS

func main() {
	config := &viessmann.Client{}
	mu := sync.Mutex{}
	configBytes, err := ioutil.ReadFile("config")
	if err == nil {
		err = json.Unmarshal(configBytes, config)
		if err != nil {
			log.Fatal("error unmarshalling config", err)
		}
	}

	sub, err := fs.Sub(web, "static")
	if err != nil {
		log.Fatal("error getting subdirectory", err)
	}

	tmpl, err := template.ParseFS(sub, "index.html")
	if err != nil {
		log.Fatal("error with template", err)
	}

	fileServer := http.FileServer(http.FS(sub))
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" || req.URL.Path == "/index.html" {
			mu.Lock()
			token := config.RefreshToken
			mu.Unlock()
			if token == "" {
				http.Redirect(w, req, "/login.html", http.StatusSeeOther)
			} else {
				buf := &bytes.Buffer{}
				err = tmpl.Execute(buf, config)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				_, _ = io.Copy(w, buf)
			}
		} else {
			fileServer.ServeHTTP(w, req)
		}
	})
	http.HandleFunc("/auth", func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()
		code := query.Get("code")
		if code == "" {
			clientId := query.Get("clientId")
			mu.Lock()
			config.ClientId = clientId
			mu.Unlock()
			http.Redirect(w, req, "https://iam.viessmann.com/idp/v2/authorize?client_id="+clientId+
				"&redirect_uri=http://localhost:3000/auth"+
				"&response_type=code"+
				"&code_challenge=2e21faa1-db2c-4d0b-a10f-575fd372bc8c-575fd372bc8c"+
				"&scope=IoT%20User%20offline_access", http.StatusFound)
		} else {
			data := url.Values{}
			mu.Lock()
			data.Set("client_id", config.ClientId)
			mu.Unlock()
			data.Set("redirect_uri", "http://localhost:3000/auth")
			data.Set("code_verifier", "2e21faa1-db2c-4d0b-a10f-575fd372bc8c-575fd372bc8c")
			data.Set("code", code)
			data.Set("grant_type", "authorization_code")
			res, err := http.Post("https://iam.viessmann.com/idp/v2/token",
				"application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			body, err := ioutil.ReadAll(res.Body)
			_ = res.Body.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if res.StatusCode != http.StatusOK {
				http.Error(w, string(body), res.StatusCode)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			err = json.Unmarshal(body, &config)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			configBytes, err := json.Marshal(config)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = ioutil.WriteFile("config", configBytes, 0600)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, req, "/", http.StatusFound)
		}
	})

	fmt.Println("Server listening on http://localhost:3000")
	err = http.ListenAndServe("localhost:3000", nil)
	if err != nil {
		log.Fatal("server error", err)
	}
}
