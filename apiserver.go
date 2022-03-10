package main

import (
	"errors"
	"net/http"
	"sync"
)

type API struct {
	Path        string
	HandlerFunc func(http.ResponseWriter, *http.Request, map[string]string)
	Params      map[string]string
}

type Config struct {
	HTTPServerConfig http.Server
	AuthMethod       func(http.ResponseWriter, *http.Request) bool
	APIAddChan       *chan API
}

func (c *Config) ServerListen() error {
	type LinkStruct struct {
		Func   func(http.ResponseWriter, *http.Request, map[string]string)
		Params map[string]string
	}
	var PathExist struct {
		Mu   sync.Mutex
		Link map[string]LinkStruct
	}
	PathExist.Link = make(map[string]LinkStruct)
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
		if Result, ok := PathExist.Link[r.URL.Path]; ok && Result.Func != nil {
			Result.Func(w, r, Result.Params)
		} else {
			if ResultDefault, ok := PathExist.Link["/"]; ok && Result.Func != nil {
				ResultDefault.Func(w, r, ResultDefault.Params)
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
					PathExist.Link[NewAPI.Path] = LinkStruct{
						Func:   NewAPI.HandlerFunc,
						Params: NewAPI.Params,
					}
					PathExist.Mu.Unlock()
				}
			}
		}(c.APIAddChan)
	} else {
		return errors.New("APIAddChan is nil")
	}
	if ServerConfig.TLSConfig == nil {
		return ServerConfig.ListenAndServe()
	} else {
		return ServerConfig.ListenAndServeTLS("", "")
	}
}
