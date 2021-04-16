package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	m "github.com/FleMo93/dcs-go-stats"
)

type StatsConfig struct {
	SourceDir string `json:"sourceDir"`
	OutputDir string `json:"outputDir"`
}

func main() {
	filename := filepath.Base(os.Args[0]) + ".log"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)

	configBytes, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Println(err)
		panic(err)
	}

	var config = StatsConfig{}
	if json.Unmarshal(configBytes, &config) != nil {
		log.Println(err)
		panic(err)
	}

	players, err := m.ReadData(config.SourceDir, config.OutputDir)
	if err != nil {
		log.Println(err)
		panic(err)
	}

	if err = m.WritePlayerNames(&players, config.OutputDir); err != nil {
		log.Println(err)
		panic(err)
	}
	if err = m.WriteTotalPlayTime(&players, config.OutputDir); err != nil {
		log.Println(err)
		panic(err)
	}
}
