package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"plugin"
	"strconv"
	"sync"
)

const (
	APPName    = "PadavanAPI"
	APPVersion = "v0.0.1-build-3"
	APPAuthor  = "Yaott"
)

var (
	ListenAddr          = "::"
	ListenPort   uint64 = 9012
	AuthPassword        = ""
	PluginConfig        = ""
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
		PluginConfig string
	}
	flag.BoolVar(&Args.Version, "v", false, "Show Version")
	flag.Uint64Var(&Args.Port, "p", 9012, "Set Port")
	flag.StringVar(&Args.AuthPassword, "auth", "", "Set Auth Password")
	flag.StringVar(&Args.PluginConfig, "d", "./config.json", "Set Plugin Dir")
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
	PluginTemp, err := ReadPlugin(PluginConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	PluginPool = PluginTemp
	Wait.Add(1)
	go Run(&Wait)
	Wait.Wait()
}

func ReadPlugin(Path string) ([]Plugin, error) {
	PluginPreConfig, err := PluginFileConfigRead(Path)
	if err != nil {
		return nil, err
	}
	PluginPoolTemp := make([]Plugin, 0)
	for _, v := range PluginPreConfig {
		if err != nil {
			log.Println(err)
			continue
		}
		plug, err := plugin.Open(v.Path)
		if err != nil {
			log.Println(err)
			continue
		}
		// Get Name
		PluginName, err := plug.Lookup("Name")
		if err != nil {
			log.Println(v.Path+":", "plugin name get fail")
			continue
		}
		TempName, ok := PluginName.(string)
		if !ok {
			log.Println(v.Path+":", "plugin name get fail")
			continue
		}
		// Get Version
		PluginVersion, err := plug.Lookup("Version")
		var TempVersion string
		if err == nil {
			TempVersion = "Unknown"
		} else {
			TempVersion, ok = PluginVersion.(string)
			if !ok {
				TempVersion = "Unknown"
			}
		}
		// Get Path
		PluginPath, err := plug.Lookup("Path")
		if err != nil {
			log.Println(err)
			continue
		}
		TempPath, ok := PluginPath.(string)
		if !ok {
			log.Println(TempName+":", "path get fail")
			continue
		}
		// Get Func
		PluginFunc, err := plug.Lookup("Handler")
		if err != nil {
			log.Println(err)
			continue
		}
		TempFunc, ok := PluginFunc.(func(http.ResponseWriter, *http.Request, map[string]string))
		if !ok {
			log.Println(TempName+":", "func get fail")
			continue
		}
		PluginPoolTemp = append(PluginPoolTemp, Plugin{
			Name:    TempName,
			Version: TempVersion,
			Path:    TempPath,
			Func:    TempFunc,
			Params:  v.Params,
		})
	}
	if len(PluginPoolTemp) <= 0 {
		return nil, errors.New("plugin not found")
	}
	return PluginPoolTemp, nil
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
