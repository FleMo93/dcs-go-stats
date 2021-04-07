package dcsgostats

import (
	"errors"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"
)

type PlayerEventType string

const (
	Connect      PlayerEventType = "Connect"
	Disconnect                   = "Disconnect"
	Kill                         = "Kill"
	KilledBy                     = "KilledBy"
	SelfKill                     = "SelfKill"
	ChangeSlot                   = "ChangeSlot"
	Crash                        = "Crash"
	Eject                        = "Eject"
	Takeoff                      = "Takeoff"
	Landing                      = "Landing"
	PilotDeath                   = "PilotDeath"
	FriendlyFire                 = "FriendlyFire"
)

type PlayerEvent struct {
	time      int64
	eventType PlayerEventType
	args      []string
}

type PlayerSession struct {
	fileName     string
	missionName  string
	sessionStart time.Time
	events       []PlayerEvent
}

type PlayerNameInfo struct {
	name   string
	occure int64
}

type Player struct {
	playerId string
	nameInfo PlayerNameInfo
	sessions []PlayerSession
}

type FileNameInfo struct {
	sessionStart int64
	missionName  string
	playerName   string
	playerId     string
}

func getEventFromString(str string) (PlayerEvent, error) {
	columns := strings.Split(str, ";")
	eventTime, err := strconv.ParseInt(columns[0], 10, 64)
	if err != nil {
		return PlayerEvent{}, err
	}

	var eventType PlayerEventType

	switch columns[1] {
	case "connect":
		eventType = Connect
	case "disconnect":
		eventType = Disconnect
	case "kill":
		eventType = Kill
	case "killed_by":
		eventType = KilledBy
	case "self_kill":
		eventType = SelfKill
	case "change_slot":
		eventType = ChangeSlot
	case "crash":
		eventType = Crash
	case "eject":
		eventType = Eject
	case "takeoff":
		eventType = Takeoff
	case "landing":
		eventType = Landing
	case "pilot_death":
		eventType = PilotDeath
	case "friendly_fire":
		eventType = FriendlyFire
	default:
		return PlayerEvent{}, errors.New("Unknown event '" + columns[1] + "'")
	}

	playerEvent := PlayerEvent{
		time:      eventTime,
		eventType: eventType,
		args:      columns[2:],
	}

	return playerEvent, nil
}

func getPlayerEvents(filePath string) ([]PlayerEvent, error) {
	statsBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return []PlayerEvent{}, err
	}

	eventLines := strings.Split(string(statsBytes), "\n")
	events := []PlayerEvent{}

	for _, line := range eventLines {
		if line == "" {
			continue
		}
		playerEvent, err := getEventFromString(line)
		if err != nil {
			return []PlayerEvent{}, err
		}

		events = append(events, playerEvent)
	}

	return events, nil
}

func getInfoFromFileName(fileName string) (FileNameInfo, error) {
	sessionInfo := strings.Split(fileName, "-[")
	for i, _ := range sessionInfo {
		sessionInfo[i] = strings.TrimSuffix(sessionInfo[i], "]")
		sessionInfo[i] = strings.TrimSuffix(sessionInfo[i], "].csv")
	}

	sessionStart, err := strconv.ParseInt(sessionInfo[0], 10, 64)
	if err != nil {
		return FileNameInfo{}, err
	}

	return FileNameInfo{
		sessionStart: sessionStart,
		missionName:  sessionInfo[1],
		playerName:   sessionInfo[2],
		playerId:     sessionInfo[3],
	}, nil
}

func ReadData(sourceDir string, statsDir string) ([]Player, error) {
	fis, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return []Player{}, err
	}

	players := make(map[string]Player)

	for _, fi := range fis {
		info, err := getInfoFromFileName(fi.Name())
		if err != nil {
			return []Player{}, nil
		}

		player, playerExists := players[info.playerId]

		if !playerExists {
			player = Player{
				playerId: info.playerId,
				nameInfo: PlayerNameInfo{
					name:   info.playerName,
					occure: info.sessionStart,
				},
				sessions: []PlayerSession{},
			}
			players[info.playerId] = player
		}

		if player.nameInfo.occure < info.sessionStart {
			player.nameInfo = PlayerNameInfo{
				name:   info.playerName,
				occure: info.sessionStart,
			}
		}

		events, err := getPlayerEvents(path.Join(sourceDir, fi.Name()))
		if err != nil {
			return []Player{}, err
		}

		playerSession := PlayerSession{
			fileName:     path.Join(sourceDir, fi.Name()),
			missionName:  info.missionName,
			sessionStart: time.Unix(info.sessionStart, 0),
			events:       events,
		}
		player.sessions = append(player.sessions, playerSession)
		players[info.playerId] = player
	}

	playerSlice := []Player{}
	for _, playerStats := range players {
		playerSlice = append(playerSlice, playerStats)
	}

	return playerSlice, nil
}

type EventConnect struct {
	*PlayerEvent
}

type EventDisconnect struct {
	*PlayerEvent
}

type EventKill struct {
	*PlayerEvent
}

type EventKilledBy struct {
	*PlayerEvent
}

type EventSelfKill struct {
	*PlayerEvent
}

type EventChangeSlot struct {
	*PlayerEvent
}

type EventCrash struct {
	*PlayerEvent
}

type EventEject struct {
	*PlayerEvent
}

type EventTakeoff struct {
	*PlayerEvent
}

type EventLanding struct {
	*PlayerEvent
}

type EventPilotDeath struct {
	*PlayerEvent
}

type EventFriendlyFire struct {
	*PlayerEvent
}

const invalidEventMessage = "Invalid event"

func ToConnectEvent(playerEvent PlayerEvent) (EventConnect, error) {
	if len(playerEvent.args) > 0 || playerEvent.eventType != Connect {
		return EventConnect{}, errors.New(invalidEventMessage)
	}

	return EventConnect{
		PlayerEvent: &playerEvent,
	}, nil
}

func ToDisconnectEvent(playerEvent PlayerEvent) (EventDisconnect, error) {
	if len(playerEvent.args) > 0 || playerEvent.eventType != Disconnect {
		return EventDisconnect{}, errors.New(invalidEventMessage)
	}

	return EventDisconnect{
		PlayerEvent: &playerEvent,
	}, nil
}
