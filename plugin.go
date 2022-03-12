package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
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
		Version: "v0.0.1-build-4",
		Path:    "/getwifiinfo",
		Func: func(w http.ResponseWriter, r *http.Request, m map[string]string) {
			if m["url"] == "" {
				w.WriteHeader(503)
				w.Write([]byte(`fail to get wifi info`))
				return
			}
			GetData := func(UrlPage string) ([]string, error) {
				resp, err := http.Get(m["url"] + UrlPage)
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
						TempSlice = append(TempSlice, WlanDrvInfoStruct{
							MAC:            TempInside[0],
							BW:             TempInside[2],
							TransportSpeed: TempInside[7],
							RSSI:           TempInside[8],
							ConnectedTime:  TempInside[10],
						})
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
						TempSlice = append(TempSlice, WlanDrvInfoStruct{
							MAC:            TempInside[0],
							BW:             TempInside[2],
							TransportSpeed: TempInside[7],
							RSSI:           TempInside[8],
							ConnectedTime:  TempInside[10],
						})
					}
					Data["Guest"] = TempSlice
				} else {
					Data["Guest"] = nil
				}
				return Data
			}
			var wg sync.WaitGroup
			GetMoreInfo := func(DataList *map[string][]WlanDrvInfoStruct) {
				wg.Add(1)
				defer wg.Done()
				GetNameAndIPMain := func(MAC string) (string, string) {
					respInside, err := http.Get(strings.ReplaceAll(m["main_get_info"], "%M", MAC))
					if err != nil {
						return "", ""
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
				for k, v := range *DataList {
					switch k {
					case "Main":
						if v == nil || len(v) == 0 {
							continue
						} else {
							if _, ok := m["main_get_info"]; ok {
								for _, v2 := range v {
									NameGet, IPGet := GetNameAndIPMain(v2.MAC)
									v2.DevName = NameGet
									v2.IP = IPGet
								}
							}
						}
					case "Guest":
						if v == nil || len(v) == 0 {
							continue
						} else {
							for _, v2 := range v {
								v2.DevName = "*"
								v2.IP = "*"
							}
						}
					default:
						continue
					}
				}

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
			go GetMoreInfo(&Result2)
			go GetMoreInfo(&Result5)
			wg.Wait()
			ResultJSON := make(map[string]interface{})
			ResultJSON["2.4g"] = Result2
			ResultJSON["5g"] = Result5
			ResultJSONReal, _ := json.Marshal(ResultJSON)
			w.Write(ResultJSONReal)
		},
		Params: Config.GetWifiInfo,
	}
	PluginTemp = append(PluginTemp, PluginPreAdd)
	PluginPreAdd = Plugin{}
	//
	return PluginTemp
}
