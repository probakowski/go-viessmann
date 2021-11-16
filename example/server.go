package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/probakowski/go-viessmann"
	"html/template"
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

type Config struct {
	RefreshToken string `json:"refresh_token"`
	ClientId     string `json:"client_id"`
	AccessToken  string `json:"access_token"`
}

func main() {
	config := &viessmann.Api{}
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
				buf := bytes.NewBufferString("")
				err = tmpl.Execute(buf, config)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				_, _ = w.Write(buf.Bytes())
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
			auth := Config{}
			err = json.Unmarshal(body, &auth)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			mu.Lock()
			config.RefreshToken = auth.RefreshToken
			configBytes, _ := json.Marshal(config)
			mu.Unlock()
			_ = ioutil.WriteFile("config", configBytes, 0600)
			http.Redirect(w, req, "/", http.StatusFound)
		}
	})

	fmt.Println("Server listening on http://localhost:3000")
	err = http.ListenAndServe("localhost:3000", nil)
	if err != nil {
		log.Fatal("server error", err)
	}
}
