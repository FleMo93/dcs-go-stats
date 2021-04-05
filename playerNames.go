package dcsgostats

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

func WritePlayerNames(players *[]Player, targetDir string) error {
	os.Mkdir(targetDir, os.ModePerm)
	playerNames := make(map[string]string)

	for _, player := range *players {
		playerNames[player.playerId] = player.nameInfo.name
	}

	jsonBytes, err := json.Marshal(playerNames)
	if err != nil {
		return err
	}

	ioutil.WriteFile(
		path.Join(targetDir, "player-names.json"),
		jsonBytes,
		0644)

	return nil
}
