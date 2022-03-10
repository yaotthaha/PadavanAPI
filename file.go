package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
)

type PluginConfigStruct struct {
	Path   string
	Params map[string]string
}

func PluginFileConfigRead(FileName string) ([]PluginConfigStruct, error) {
	File, err := os.Open(FileName)
	if err != nil {
		return nil, err
	}
	defer func(File *os.File) {
		_ = File.Close()
	}(File)
	DataRaw, err := ioutil.ReadAll(File)
	if err != nil {
		return nil, err
	}
	var Data []PluginConfigStruct
	err = json.Unmarshal(DataRaw, &Data)
	if err != nil {
		return nil, err
	}
	if len(Data) <= 0 {
		return nil, errors.New("plugin not found")
	}
	return Data, nil
}
