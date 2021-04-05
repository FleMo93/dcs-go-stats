package main

import (
	"encoding/json"
	"io/ioutil"

	m "github.com/FleMo93/dcs-go-stats"
)

type StatsConfig struct {
	SourceDir string `json:"sourceDir"`
	OutputDir string `json:"outputDir"`
}

func main() {
	configBytes, err := ioutil.ReadFile("./config.json")
	if err != nil {
		panic(err)
	}

	var config = StatsConfig{}
	if json.Unmarshal(configBytes, &config) != nil {
		panic(err)
	}

	players, err := m.ReadData(config.SourceDir, config.OutputDir)
	if err != nil {
		panic(err)
	}

	if err = m.WritePlayerNames(&players, config.OutputDir); err != nil {
		panic(err)
	}
	if err = m.WriteTotalPlayTime(&players, config.OutputDir); err != nil {
		panic(err)
	}
}
