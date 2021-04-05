package dcsgostats

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

const handleFileError = "Could not handle play time file \"{0}\":"

func WriteTotalPlayTime(players *[]Player, targetDir string) error {
	os.Mkdir(targetDir, os.ModePerm)
	playTimes := make(map[string]int64)

	for _, player := range *players {
		var playTime int64 = 0
		for _, session := range player.sessions {
			conEv, err := ToConnectEvent(session.events[0])
			if err != nil {
				log.Println(strings.Replace(handleFileError, "{0}", session.fileName, 1))
				log.Println(err)
				continue
			}

			disConEv, err := ToDisconnectEvent(session.events[len(session.events)-1])
			if err != nil {
				log.Println(strings.Replace(handleFileError, "{0}", session.fileName, 1))
				log.Println(err)
				continue
			}

			playTime += disConEv.time - conEv.time
		}

		playTimes[player.playerId] = playTime
	}

	jsonBytes, err := json.Marshal(playTimes)
	if err != nil {
		return err
	}

	ioutil.WriteFile(
		path.Join(targetDir, "total-times.json"),
		jsonBytes,
		0644)

	return nil
}
