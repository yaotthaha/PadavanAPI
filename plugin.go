package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"os"
	"strings"
)

type ConfigStruct struct {
	GetWifiInfo map[string]string `json:"get_wifi_info"`
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
		Version: "v0.0.1-build-1",
		Path:    "/getwifiinfo",
		Func: func(w http.ResponseWriter, r *http.Request, m map[string]string) {
			if m["url"] == "" {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			resp, err := http.Get(m["url"])
			if err != nil {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			doc, err := goquery.NewDocumentFromReader(resp.Body)
			if err != nil {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			var msg string
			doc.Find(`textarea`).Each(func(i int, selection *goquery.Selection) {
				msg = selection.Text()
			})
			msgSlice := strings.Split(msg, "\n")
			type WlanDrvInfoStruct struct {
				MAC            string
				BW             string
				TransportSpeed string
				RSSI           string
				ConnectedTime  string
			}
			type WlanDrvStruct struct {
				Main  []WlanDrvInfoStruct
				Guest []WlanDrvInfoStruct
			}
			Result := func() WlanDrvStruct {
				var (
					MainStart  = 0
					MainEnd    = 0
					GuestTag   = false
					GuestStart = 0
					GuestEnd   = 0
				)
				for k, v := range msgSlice {
					switch v {
					case `AP Main Stations List`:
						MainStart = k + 3
					case `AP Guest Stations List`:
						GuestTag = true
						MainEnd = k - 2
						GuestStart = k + 3
						GuestEnd = len(msgSlice)
					default:
						continue
					}
				}
				if !GuestTag {
					MainEnd = len(msgSlice) - 1
				}
				WlanDrv := WlanDrvStruct{}
				if GuestTag {
					TempSlice1 := make([]WlanDrvInfoStruct, 0)
					TempSlice2 := make([]WlanDrvInfoStruct, 0)
					for _, v := range msgSlice[MainStart : MainEnd+1] {
						Info := make([]string, 0)
						for _, v2 := range strings.Split(v, " ") {
							if v2 == "" {
								continue
							}
							Info = append(Info, v2)
						}
						TempSlice1 = append(TempSlice1, WlanDrvInfoStruct{
							MAC:            Info[0],
							BW:             Info[2],
							TransportSpeed: Info[7],
							RSSI:           Info[8],
							ConnectedTime:  Info[10],
						})
					}
					for _, v := range msgSlice[GuestStart : GuestEnd-1] {
						Info := make([]string, 0)
						for _, v2 := range strings.Split(v, " ") {
							if v2 == "" {
								continue
							}
							Info = append(Info, v2)
						}
						TempSlice2 = append(TempSlice2, WlanDrvInfoStruct{
							MAC:            Info[0],
							BW:             Info[2],
							TransportSpeed: Info[7],
							RSSI:           Info[8],
							ConnectedTime:  Info[10],
						})
					}
					WlanDrv.Main = TempSlice1
					WlanDrv.Guest = TempSlice2
				} else {
					TempSlice := make([]WlanDrvInfoStruct, 0)
					for _, v := range msgSlice[MainStart : MainEnd+1] {
						Info := make([]string, 0)
						for _, v2 := range strings.Split(v, " ") {
							if v2 == "" {
								continue
							}
							Info = append(Info, v2)
						}
						TempSlice = append(TempSlice, WlanDrvInfoStruct{
							MAC:            Info[0],
							BW:             Info[2],
							TransportSpeed: Info[7],
							RSSI:           Info[8],
							ConnectedTime:  Info[10],
						})
					}
					WlanDrv.Main = TempSlice
				}
				return WlanDrv
			}()
			ResultJSON, err := json.Marshal(Result)
			if err != nil {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			w.Write(ResultJSON)
		},
		Params: Config.GetWifiInfo,
	}
	PluginTemp = append(PluginTemp, PluginPreAdd)
	PluginPreAdd = Plugin{}
	//
	return PluginTemp
}
