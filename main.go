package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"plugin"
	"strconv"
)

const (
	APPName    = "PadavanAPI"
	APPVersion = "v0.0.1-build-1"
	APPAuthor  = "Yaott"
)

var (
	ListenAddr          = "::"
	ListenPort   uint64 = 9012
	AuthPassword        = ""
	PluginDir           = ""
)

type Plugin struct {
	Path string
	Func func(http.ResponseWriter, *http.Request)
}

var (
	APIAddChannel chan API
	PluginPool    []Plugin
)

func main() {
	var Args struct {
		Version      bool
		Port         uint64
		AuthPassword string
		PluginDir    string
	}
	flag.BoolVar(&Args.Version, "v", false, "Show Version")
	flag.Uint64Var(&Args.Port, "p", 9012, "Set Port")
	flag.StringVar(&Args.AuthPassword, "auth", "", "Set Auth Password")
	flag.StringVar(&Args.PluginDir, "d", "./", "Set Plugin Dir")
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
	var err error
	PluginDir, err = filepath.Abs(Args.PluginDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid plugin dir")
		return
	}
	if err = ReadPlugin(PluginDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	Run()
}

func ReadPlugin(Dir string) error {
	PluginPre := GetDirAllFileName(Dir, `^(.*).so$`)
	if len(PluginPre) <= 0 {
		return errors.New(Dir + " : plugin not found")
	}
	PluginPool = make([]Plugin, 0)
	for _, v := range PluginPre {
		plug, err := plugin.Open(v)
		if err != nil {
			log.Println(err)
			continue
		}
		PluginPath, err := plug.Lookup("Path")
		if err != nil {
			log.Println(err)
			continue
		}
		TempPath, ok := PluginPath.(string)
		if !ok {
			log.Println(v+":", "path get fail")
			continue
		}
		PluginFunc, err := plug.Lookup("Handler")
		if err != nil {
			log.Println(err)
			continue
		}
		TempFunc, ok := PluginFunc.(func(http.ResponseWriter, *http.Request))
		if !ok {
			log.Println(v+":", "func get fail")
			continue
		}
		PluginPool = append(PluginPool, Plugin{
			Path: TempPath,
			Func: TempFunc,
		})
	}
	if len(PluginPool) <= 0 {
		return errors.New("plugin not found")
	}
	return nil
}

func Run() {
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
		HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok\n"))
			return
		},
	}
	for _, v := range PluginPool {
		APIAddChannel <- API{
			Path:        v.Path,
			HandlerFunc: v.Func,
		}
	}
	fmt.Fprintln(os.Stderr, Config.ServerListen())
}
