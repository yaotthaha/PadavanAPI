package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ConfigStruct struct {
	GetWifiInfo map[string]string `json:"get_wifi_info"`
	GetSysInfo  map[string]string `json:"get_sys_info"`
}

func AddPlugin(ConfigFile string) []Plugin {
	PluginTemp := make([]Plugin, 0)
	var Config ConfigStruct
	err := ReadFile(ConfigFile, &Config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fail to read config file")
	}
	var PluginPreAdd Plugin
	// Get Wifi Info
	PluginPreAdd = Plugin{
		Name:    "GetWifiInfo",
		Version: "v0.0.1-build-6",
		Path:    "/getwifiinfo",
		Func: func(w http.ResponseWriter, r *http.Request, m map[string]string) {
			if m["url"] == "" {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			GetData := func(UrlPage string) ([]string, error) {
				req, err := http.NewRequest(http.MethodGet, m["url"]+UrlPage, nil)
				if err != nil {
					return nil, errors.New(`fail to get wifi info`)
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return nil, errors.New(`fail to get wifi info`)
				}
				doc, err := goquery.NewDocumentFromReader(resp.Body)
				if err != nil {
					return nil, errors.New(`fail to get wifi info`)
				}
				var msg string
				doc.Find(`textarea`).Each(func(i int, selection *goquery.Selection) {
					msg = selection.Text()
				})
				msgSlice := strings.Split(msg, "\n")
				return msgSlice, nil
			}
			type WlanDrvInfoStruct struct {
				DevName        string
				IP             string
				MAC            string
				BW             string
				TransportSpeed string
				RSSI           string
				ConnectedTime  string
			}
			GetNameAndIPMain := func(MAC string) (string, string) {
				respInside, err := http.Get(strings.ReplaceAll(m["main_get_info"], "%M", MAC))
				if err != nil {
					return "*", "*"
				}
				D, err := ioutil.ReadAll(respInside.Body)
				if err != nil {
					return "*", "*"
				}
				DSlice := strings.Split(string(D), " ")
				Name := DSlice[0][1 : len(DSlice[0])-1]
				IP := DSlice[1][1 : len(DSlice[1])-1]
				return Name, IP
			}
			Result := func(DataRaw []string) map[string][]WlanDrvInfoStruct {
				Slices := make([][]string, 0)
				var (
					TempSlice []string
					RunTag    bool
				)
				for _, v := range DataRaw {
					if v != "" {
						if RunTag {
							TempSlice = append(TempSlice, v)
						} else {
							TempSlice = nil
							TempSlice = make([]string, 0)
							TempSlice = append(TempSlice, v)
							RunTag = true
						}
					} else {
						if RunTag {
							Slices = append(Slices, TempSlice)
							TempSlice = nil
							RunTag = false
						}
					}
				}
				var (
					MainTag  = 0
					GuestTag = 0
				)
				for k, v := range Slices {
					switch v[0] {
					case "AP Main Stations List":
						if len(v) <= 3 {
							MainTag = -1
						} else {
							MainTag = k
						}
					case "AP Guest Stations List":
						if len(v) <= 3 {
							GuestTag = -1
						} else {
							GuestTag = k
						}
					default:
						continue
					}
				}
				Data := make(map[string][]WlanDrvInfoStruct)
				RemoveNil := func(Slice []string) []string {
					Temp := make([]string, 0)
					for _, v := range Slice {
						if v != "" {
							Temp = append(Temp, v)
						}
					}
					return Temp
				}
				if MainTag < 0 {
					Data["Main"] = []WlanDrvInfoStruct{}
				} else if MainTag > 0 {
					TempSlice := make([]WlanDrvInfoStruct, 0)
					for _, v := range Slices[MainTag][3:] {
						TempInside := RemoveNil(strings.Split(v, " "))
						TempInfo := WlanDrvInfoStruct{
							MAC:            TempInside[0],
							BW:             TempInside[2],
							TransportSpeed: TempInside[7],
							RSSI:           TempInside[8],
							ConnectedTime:  TempInside[10],
						}
						if _, ok := m["main_get_info"]; ok {
							NameGet, IPGet := GetNameAndIPMain(TempInfo.MAC)
							TempInfo.DevName = NameGet
							TempInfo.IP = IPGet
						}
						TempSlice = append(TempSlice, TempInfo)
					}
					Data["Main"] = TempSlice
				} else {
					Data["Main"] = nil
				}
				if GuestTag < 0 {
					Data["Guest"] = []WlanDrvInfoStruct{}
				} else if GuestTag > 0 {
					TempSlice := make([]WlanDrvInfoStruct, 0)
					for _, v := range Slices[GuestTag][3:] {
						TempInside := RemoveNil(strings.Split(v, " "))
						TempInfo := WlanDrvInfoStruct{
							MAC:            TempInside[0],
							BW:             TempInside[2],
							TransportSpeed: TempInside[7],
							RSSI:           TempInside[8],
							ConnectedTime:  TempInside[10],
						}
						TempInfo.DevName = "*"
						TempInfo.IP = "*"
						TempSlice = append(TempSlice, TempInfo)
					}
					Data["Guest"] = TempSlice
				} else {
					Data["Guest"] = nil
				}
				return Data
			}
			GetData2, err := GetData("/Main_WStatus2g_Content.asp")
			if err != nil {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			Result2 := Result(GetData2)
			GetData5, err := GetData("/Main_WStatus_Content.asp")
			if err != nil {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			Result5 := Result(GetData5)
			ResultJSON := make(map[string]interface{})
			ResultJSON["2.4g"] = Result2
			ResultJSON["5g"] = Result5
			ResultJSONReal, _ := json.Marshal(ResultJSON)
			w.Write(ResultJSONReal)
		},
		Params: Config.GetWifiInfo,
	}
	PluginTemp = append(PluginTemp, PluginPreAdd)
	// Get System Info
	PluginPreAdd = Plugin{
		Name:    "GetSysInfo",
		Version: "v0.0.1-build-1",
		Path:    "/getsysinfo",
		Func: func(w http.ResponseWriter, r *http.Request, m map[string]string) {
			// RAM
			GetMem := func() map[string]string {
				File, err := os.Open("/proc/meminfo")
				if err != nil {
					return nil
				}
				DataRaw, err := ioutil.ReadAll(File)
				if err != nil {
					return nil
				}
				DataSlice := strings.Split(string(DataRaw), "\n")
				DataMap := make(map[string]string)
				DO := func(Value string) string {
					var TempTag = ""
					for _, v := range strings.Split(Value, " ") {
						if v != "" {
							if TempTag == "" {
								TempTag = "nil"
								continue
							} else {
								TempTag = v
								break
							}
						}
					}
					TempN, _ := strconv.Atoi(TempTag)
					return fmt.Sprintf("%v MB", math.Trunc(float64(float32(TempN)/1024)*1e2+0.5)*1e-2)
				}
				DataMap["MemTotal"] = DO(DataSlice[0])
				DataMap["MemFree"] = DO(DataSlice[1])
				DataMap["MemAvailable"] = DO(DataSlice[2])
				return DataMap
			}()
			// Load Average
			GetLoadAvg := func() []string {
				File, err := os.Open("/proc/loadavg")
				if err != nil {
					return nil
				}
				DataRaw, err := ioutil.ReadAll(File)
				if err != nil {
					return nil
				}
				DataSlice := strings.Split(string(DataRaw), " ")
				return []string{DataSlice[0], DataSlice[1], DataSlice[2]}
			}()
			DataMap := make(map[string]interface{})
			DataMap["Mem"] = GetMem
			DataMap["LoadAvg"] = GetLoadAvg
			DataJson, err := json.Marshal(DataMap)
			if err != nil {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get system info`))
				return
			}
			_, _ = w.Write(DataJson)
		},
		Params: Config.GetWifiInfo,
	}
	PluginTemp = append(PluginTemp, PluginPreAdd)
	// Ban Wifi Dev
	PluginPreAdd = Plugin{
		Name:    "BanWifiDev",
		Version: "v0.0.1-build-1",
		Path:    "/banwifidev",
		Func: func(w http.ResponseWriter, r *http.Request, m map[string]string) {
			ParamsMap := func() map[string]string {
				ParamsSlice := make([]string, 0)
				for _, v := range strings.Split(r.URL.RawQuery, "&") {
					ParamsSlice = append(ParamsSlice, v)
				}
				PostDataRaw := make([]byte, 0)
				PostDataRaw, err := ioutil.ReadAll(r.Body)
				if err == nil {
					var PostDataMap map[string]string
					err = json.Unmarshal(PostDataRaw, &PostDataMap)
					if err == nil {
						for k, v := range PostDataMap {
							ParamsSlice = append(ParamsSlice, k+"="+v)
						}
					}
				}
				ParamsMapTemp := make(map[string]string)
				for _, v := range ParamsSlice {
					TempInside := strings.Split(v, "=")
					ParamsMapTemp[TempInside[0]] = TempInside[1]
				}
				return ParamsMapTemp
			}()
			var (
				Dev string
				MAC string
				ok  bool
			)
			if Dev, ok = ParamsMap["dev"]; !ok {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get dev info`))
				return
			}
			if MAC, ok = ParamsMap["mac"]; !ok {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get mac info`))
				return
			}
			cmd := exec.Command("/bin/sh", "-c", "/bin/iwpriv "+Dev+" set DisConnectSta="+MAC)
			cmd.Stdout = nil
			cmd.Stderr = nil
			err := cmd.Run()
			if err == nil {
				w.WriteHeader(200)
			} else {
				w.WriteHeader(503)
				w.Write([]byte(`fail to run`))
			}
			return
		},
		Params: nil,
	}
	PluginTemp = append(PluginTemp, PluginPreAdd)
	return PluginTemp
}
