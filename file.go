package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

func ReadFile(FileName string, ConfigWrite *ConfigStruct) error {
	File, err := os.Open(FileName)
	if err != nil {
		return err
	}
	defer func(File *os.File) {
		_ = File.Close()
	}(File)
	DataBytes, err := ioutil.ReadAll(File)
	if err != nil {
		return err
	}
	return json.Unmarshal(DataBytes, ConfigWrite)
}
