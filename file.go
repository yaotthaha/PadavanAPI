package main

import (
	"io/ioutil"
	"path"
	"regexp"
	"sync"
)

func GetDirAllFileName(Dir string, RegexpPattern string) []string {
	FileNamePool := make(chan string, 512)
	FileNameSlice := make([]string, 0)
	var wg sync.WaitGroup
	wg.Add(1)
	go DirReadAll(Dir, RegexpPattern, &FileNamePool, &wg)
	wg.Wait()
	for len(FileNamePool) > 0 {
		FileNameSlice = append(FileNameSlice, <-FileNamePool)
	}
	return FileNameSlice
}

func DirReadAll(DirInside string, RegexpPattern string, InputChan *chan string, WaitGroup *sync.WaitGroup) {
	defer WaitGroup.Done()
	fileInfo, err := ioutil.ReadDir(DirInside)
	if err != nil {
		return
	}
	if len(fileInfo) <= 0 {
		return
	}
	for _, v := range fileInfo {
		if v.IsDir() {
			WaitGroup.Add(1)
			go DirReadAll(path.Join(DirInside, v.Name()), RegexpPattern, InputChan, WaitGroup)
		} else if match, _ := regexp.MatchString(RegexpPattern, v.Name()); match {
			*InputChan <- path.Join(DirInside, v.Name())
		}
	}
}
