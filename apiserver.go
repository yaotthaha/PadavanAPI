package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
)

type API struct {
	Path        string
	HandlerFunc func(http.ResponseWriter, *http.Request)
}

type Config struct {
	HTTPServerConfig http.Server
	AuthMethod       func(http.ResponseWriter, *http.Request) bool
	APIAddChan       *chan API
	APIDelChan       *chan string
}

func HttpGetParams(r *http.Request) map[string]string {
	ArgsALL := make(map[string]string)
	arg := r.URL.Query()
	for k, v := range arg {
		ArgsALL[k] = v[len(v)-1]
	}
	DataPost, _ := ioutil.ReadAll(r.Body)
	DataPostMap := make(map[string]string)
	_ = json.Unmarshal(DataPost, &DataPostMap)
	for k, v := range DataPostMap {
		ArgsALL[k] = v
	}
	return ArgsALL
}

func (c *Config) ServerListen() error {
	var PathExist struct {
		Mu   sync.Mutex
		Link map[string]func(http.ResponseWriter, *http.Request)
	}
	PathExist.Link = make(map[string]func(http.ResponseWriter, *http.Request))
	Mux := http.NewServeMux()
	ServerConfig := &c.HTTPServerConfig
	ServerConfig.Handler = Mux
	Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if c.AuthMethod != nil {
			if !c.AuthMethod(w, r) {
				return
			}
		}
		PathExist.Mu.Lock()
		defer PathExist.Mu.Unlock()
		if Result, ok := PathExist.Link[r.URL.Path]; ok && Result != nil {
			Result(w, r)
		} else {
			if ResultDefault, ok := PathExist.Link["/"]; ok && Result != nil {
				ResultDefault(w, r)
			} else {
				w.WriteHeader(503)
			}
		}
	})
	if c.APIAddChan != nil {
		go func(Chan *chan API) {
			for {
				select {
				case NewAPI := <-*Chan:
					if NewAPI.Path[:1] != "/" {
						NewAPI.Path = "/" + NewAPI.Path
					}
					PathExist.Mu.Lock()
					PathExist.Link[NewAPI.Path] = NewAPI.HandlerFunc
					PathExist.Mu.Unlock()
				}
			}
		}(c.APIAddChan)
	} else {
		return errors.New("APIAddChan is nil")
	}
	if c.APIDelChan != nil {
		go func(Chan *chan string) {
			for {
				select {
				case APIPathDel := <-*Chan:
					PathExist.Mu.Lock()
					if _, ok := PathExist.Link[APIPathDel]; ok {
						delete(PathExist.Link, APIPathDel)
					}
					PathExist.Mu.Unlock()
				}
			}
		}(c.APIDelChan)
	}
	if ServerConfig.TLSConfig == nil {
		return ServerConfig.ListenAndServe()
	} else {
		return ServerConfig.ListenAndServeTLS("", "")
	}
}
