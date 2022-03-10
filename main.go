package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
)

const (
	APPName    = "PadavanAPI"
	APPVersion = "v0.0.1-build-4"
	APPAuthor  = "Yaott"
)

var (
	ListenAddr          = "::"
	ListenPort   uint64 = 9012
	AuthPassword        = ""
)

type Plugin struct {
	Name    string
	Version string
	Path    string
	Func    func(http.ResponseWriter, *http.Request, map[string]string)
	Params  map[string]string
}

var (
	APIAddChannel chan API
	PluginPool    []Plugin
	Wait          sync.WaitGroup
)

func main() {
	var Args struct {
		Version      bool
		Port         uint64
		AuthPassword string
		Config       string
	}
	flag.BoolVar(&Args.Version, "v", false, "Show Version")
	flag.Uint64Var(&Args.Port, "p", 9012, "Set Port")
	flag.StringVar(&Args.AuthPassword, "auth", "", "Set Auth Password")
	flag.StringVar(&Args.Config, "c", "./config.json", "Set Plugin Config")
	flag.Parse()
	if Args.Version {
		fmt.Fprintln(os.Stdout, APPName+"/"+APPVersion, "Build From", APPAuthor)
		return
	}
	if Args.Port == 0 || Args.Port > 65535 {
		fmt.Fprintln(os.Stderr, "invalid port")
		return
	} else {
		ListenPort = Args.Port
	}
	AuthPassword = Args.AuthPassword
	PluginPool = AddPlugin(Args.Config)
	Wait.Add(1)
	go Run(&Wait)
	Wait.Wait()
}

func Run(WaitGroup *sync.WaitGroup) {
	defer WaitGroup.Done()
	APIAddChannel = make(chan API, 10)
	Config := Config{
		HTTPServerConfig: http.Server{
			Addr:     net.JoinHostPort(ListenAddr, strconv.FormatUint(ListenPort, 10)),
			ErrorLog: log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Llongfile),
		},
		AuthMethod: func(w http.ResponseWriter, r *http.Request) bool {
			if AuthPassword != "" {
				HeaderMap := r.Header
				if HeaderMap == nil {
					w.WriteHeader(403)
					return false
				} else {
					Auth := HeaderMap["Auth"]
					PassTag := false
					for _, v := range Auth {
						if v == AuthPassword {
							PassTag = true
							break
						}
					}
					if !PassTag {
						w.WriteHeader(403)
						return false
					}
					return true
				}
			} else {
				return true
			}
		},
		APIAddChan: &APIAddChannel,
	}
	APIAddChannel <- API{
		Path: "/",
		HandlerFunc: func(w http.ResponseWriter, r *http.Request, Params map[string]string) {
			_, _ = w.Write([]byte("ok\n"))
			return
		},
	}
	for _, v := range PluginPool {
		fmt.Fprintln(os.Stdout, "Plugin["+v.Name+"/"+v.Version+"] Path: "+v.Path)
		APIAddChannel <- API{
			Path:        v.Path,
			HandlerFunc: v.Func,
			Params:      v.Params,
		}
	}
	fmt.Fprintln(os.Stderr, Config.ServerListen())
}
