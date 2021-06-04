package dcsgostats

import (
	"bufio"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type Source struct {
	Name      string `json:"name"`
	Directory string `json:"dir"`
}

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
	source       string
	events       []PlayerEvent
	sorties      []Sortie
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

type Sortie struct {
	startTime          int64
	endTime            int64
	plane              string
	endReason          string
	eventTakeoff       *EventTakeoff
	eventLanding       *EventLanding
	eventKills         []*EventKill
	eventFriendlyFires []*EventFriendlyFire
	eventEject         *EventEject
	eventPilotDeath    *EventPilotDeath
	eventKilledBy      *EventKilledBy
	eventCrash         *EventCrash
	events             []*PlayerEvent
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
	file, err := os.Open(filePath)
	if err != nil {
		return []PlayerEvent{}, err
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	events := []PlayerEvent{}

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		playerEvent, err := getEventFromString(line)
		if err != nil {
			return []PlayerEvent{}, errors.New(filePath + ":\n" + err.Error())
		}

		events = append(events, playerEvent)
	}

	return events, nil
}

func getInfoFromFileName(fileName string) (FileNameInfo, error) {
	sessionInfo := strings.Split(fileName, "-[")
	for i := range sessionInfo {
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

func determineAndSetSortieEnd(sortie *Sortie, event PlayerEventType, eventTime int64) error {
	if sortie.endTime != 0 && eventTime-30 > sortie.endTime {
		return nil
	}

	switch event {
	case SelfKill:
		sortie.endTime = eventTime
		sortie.endReason = SelfKill
		break
	case KilledBy:
		if sortie.endReason == "" || sortie.endReason != SelfKill {
			sortie.endReason = KilledBy
		}
		break
	case Crash:
		if sortie.endReason == "" ||
			(sortie.endReason != SelfKill &&
				sortie.endReason != KilledBy) {
			sortie.endReason = Crash
		}
		break
	case PilotDeath:
		if sortie.endReason == "" ||
			(sortie.endReason != SelfKill &&
				sortie.endReason != KilledBy &&
				sortie.endReason != Crash) {
			sortie.endReason = PilotDeath
		}
		break
	case Eject:
		if sortie.endReason == "" ||
			sortie.endReason == Landing ||
			sortie.endReason == Disconnect ||
			sortie.endReason == ChangeSlot {
			sortie.endReason = Eject
		}
		break
	case Landing:
		if sortie.endReason == "" {
			sortie.endReason = Landing
		}
		break
	case Disconnect:
		if sortie.endReason == "" {
			sortie.endReason = Disconnect
		}
		break
	case ChangeSlot:
		if sortie.endReason == "" {
			sortie.endReason = SelfKill
		}
		break
	default:
		return errors.New("No end event")
	}

	if sortie.endTime == 0 {
		sortie.endTime = eventTime
	}

	return nil
}

func getSortiesFromSession(session *PlayerSession) ([]Sortie, error) {
	sorties := []Sortie{}
	lastPlane := ""

	sortie := Sortie{
		events:             []*PlayerEvent{},
		startTime:          0,
		endTime:            0,
		plane:              lastPlane,
		endReason:          "",
		eventKills:         []*EventKill{},
		eventFriendlyFires: []*EventFriendlyFire{},
		eventPilotDeath:    nil,
		eventCrash:         nil,
		eventLanding:       nil,
		eventTakeoff:       nil,
	}

	for _, event := range session.events {
		switch event.eventType {
		case Connect:
			break
		case Takeoff:
			takeoffEvent, err := ToTakeoffEvent(event)
			if err != nil {
				return []Sortie{}, err
			}
			sortie.startTime = event.time
			sortie.eventTakeoff = &takeoffEvent
			break

		case Kill:
			killEvent, err := ToKillEvent(event)
			if err != nil {
				return []Sortie{}, err
			}
			sortie.eventKills = append(sortie.eventKills, &killEvent)
			break

		case FriendlyFire:
			friendlyFireEvent, err := ToFriendlyFireEvent(event)
			if err != nil {
				return []Sortie{}, err
			}

			sortie.eventFriendlyFires = append(sortie.eventFriendlyFires, &friendlyFireEvent)
			break

		case ChangeSlot:
			changeSlotEvent, err := ToChangeSlotEvent(event)
			if err != nil {
				return []Sortie{}, err
			}
			sortie.plane = changeSlotEvent.unitType
			lastPlane = changeSlotEvent.unitType
			err = determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}
			break

		case Disconnect:
			err := determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}
			break

		case Landing:
			landingEvent, err := ToLandingEvent(event)
			if err != nil {
				return []Sortie{}, err
			}

			err = determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}

			sortie.eventLanding = &landingEvent
			break

		case Crash:
			crashEvent, err := ToCrashEvent(event)
			if err != nil {
				return []Sortie{}, err
			}

			err = determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}

			sortie.eventCrash = &crashEvent
			break

		case Eject:
			ejectEvent, err := ToEjectEvent(event)
			if err != nil {
				return []Sortie{}, err
			}

			err = determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}

			sortie.eventEject = &ejectEvent
			break

		case PilotDeath:
			pilotDeathEvent, err := ToPilotDeathEvent(event)
			if err != nil {
				return []Sortie{}, err
			}

			err = determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}

			sortie.eventPilotDeath = &pilotDeathEvent
			break

		case KilledBy:
			killedByEvent, err := ToKilledByEvent(event)
			if err != nil {

			}

			err = determineAndSetSortieEnd(&sortie, event.eventType, event.time)
			if err != nil {
				return []Sortie{}, err
			}

			sortie.eventKilledBy = &killedByEvent
			break

		default:
			return []Sortie{}, errors.New("Unhandeld event " + string(event.eventType))
		}

		sortie.events = append(sortie.events, &event)
	}

	return sorties, nil
}

func ReadData(sources []Source, statsDir string) ([]Player, error) {
	playersMap := make(map[string]Player)

	for _, src := range sources {
		fis, err := ioutil.ReadDir(src.Directory)
		if err != nil {
			return []Player{}, err
		}

		for _, fi := range fis {
			info, err := getInfoFromFileName(fi.Name())
			if err != nil {
				return []Player{}, nil
			}

			player, playerExists := playersMap[info.playerId]

			if !playerExists {
				player = Player{
					playerId: info.playerId,
					nameInfo: PlayerNameInfo{
						name:   info.playerName,
						occure: info.sessionStart,
					},
					sessions: []PlayerSession{},
				}
				playersMap[info.playerId] = player
			}

			if player.nameInfo.occure < info.sessionStart {
				player.nameInfo = PlayerNameInfo{
					name:   info.playerName,
					occure: info.sessionStart,
				}
			}

			events, err := getPlayerEvents(path.Join(src.Directory, fi.Name()))
			if err != nil {
				return []Player{}, err
			}

			playerSession := PlayerSession{
				fileName:     path.Join(src.Directory, fi.Name()),
				missionName:  info.missionName,
				sessionStart: time.Unix(info.sessionStart, 0),
				events:       events,
				sorties:      []Sortie{},
				source:       src.Name,
			}
			player.sessions = append(player.sessions, playerSession)
			playerSession.sorties, err = getSortiesFromSession(&playerSession)
			if err != nil {
				return []Player{}, err
			}
			playersMap[info.playerId] = player
		}
	}

	playerSlice := []Player{}
	for _, playerStats := range playersMap {
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
	killerUnitType string
	killerSide     int
	victimPlayerID string
	victimUnitType string
	victimSide     int
	weaponName     string
}

type EventKilledBy struct {
	*PlayerEvent
	killerPlayerID string
	killerUnitType string
	killerSide     int
	victimPlayerID string
	victimUnitType string
	victimSide     int
	weaponName     string
}

type EventSelfKill struct {
	*PlayerEvent
}

type EventChangeSlot struct {
	*PlayerEvent
	side      int
	unitID    string
	unitType  string
	role      string
	groupName string
}

type EventCrash struct {
	*PlayerEvent
	unitID string
}

type EventEject struct {
	*PlayerEvent
	unitID string
}

type EventTakeoff struct {
	*PlayerEvent
	airdomeName *string
	unitID      string
}

type EventLanding struct {
	*PlayerEvent
	unitID      string
	airdomeName string
}

type EventPilotDeath struct {
	*PlayerEvent
	unitID string
}

type EventFriendlyFire struct {
	*PlayerEvent
	weaponName     string
	victimPlayerID string
}

const invalidEventMessage = "Invalid event: "

func ToConnectEvent(playerEvent PlayerEvent) (EventConnect, error) {
	if len(playerEvent.args) > 0 || playerEvent.eventType != Connect {
		return EventConnect{}, errors.New(invalidEventMessage + string(Connect))
	}

	return EventConnect{
		PlayerEvent: &playerEvent,
	}, nil
}

func ToDisconnectEvent(playerEvent PlayerEvent) (EventDisconnect, error) {
	if len(playerEvent.args) > 0 || playerEvent.eventType != Disconnect {
		return EventDisconnect{}, errors.New(invalidEventMessage + Disconnect)
	}

	return EventDisconnect{
		PlayerEvent: &playerEvent,
	}, nil
}

func ToTakeoffEvent(playerEvent PlayerEvent) (EventTakeoff, error) {
	if len(playerEvent.args) < 1 || len(playerEvent.args) > 2 || playerEvent.eventType != Takeoff {
		return EventTakeoff{}, errors.New(invalidEventMessage + Takeoff)
	}

	var airdomeName *string = nil
	if len(playerEvent.args) == 2 {
		airdomeName = &playerEvent.args[1]
	}

	return EventTakeoff{
		PlayerEvent: &playerEvent,
		unitID:      playerEvent.args[0],
		airdomeName: airdomeName,
	}, nil
}

func ToChangeSlotEvent(playerEvent PlayerEvent) (EventChangeSlot, error) {
	if len(playerEvent.args) != 5 || playerEvent.eventType != ChangeSlot {
		return EventChangeSlot{}, errors.New(invalidEventMessage + ChangeSlot)
	}

	side, err := strconv.Atoi(playerEvent.args[0])
	if err != nil {
		return EventChangeSlot{}, err
	}

	return EventChangeSlot{
		PlayerEvent: &playerEvent,
		side:        side,
		unitID:      playerEvent.args[1],
		unitType:    playerEvent.args[2],
		role:        playerEvent.args[3],
		groupName:   playerEvent.args[4],
	}, nil
}

func ToLandingEvent(playerEvent PlayerEvent) (EventLanding, error) {
	if len(playerEvent.args) != 2 || playerEvent.eventType != Landing {
		return EventLanding{}, errors.New(invalidEventMessage + Landing)
	}

	return EventLanding{
		PlayerEvent: &playerEvent,
		unitID:      playerEvent.args[0],
		airdomeName: playerEvent.args[1],
	}, nil
}

func ToCrashEvent(playerEvent PlayerEvent) (EventCrash, error) {
	if len(playerEvent.args) != 1 || playerEvent.eventType != Crash {
		return EventCrash{}, errors.New(invalidEventMessage + Crash)
	}

	return EventCrash{
		PlayerEvent: &playerEvent,
		unitID:      playerEvent.args[0],
	}, nil
}

func ToEjectEvent(playerEvent PlayerEvent) (EventEject, error) {
	if len(playerEvent.args) != 1 || playerEvent.eventType != Eject {
		return EventEject{}, errors.New(invalidEventMessage + Eject)
	}

	return EventEject{
		PlayerEvent: &playerEvent,
		unitID:      playerEvent.args[0],
	}, nil
}

func ToKillEvent(playerEvent PlayerEvent) (EventKill, error) {
	if len(playerEvent.args) != 6 || playerEvent.eventType != Kill {
		return EventKill{}, errors.New(invalidEventMessage + Kill)
	}

	killerSide, err := strconv.Atoi(playerEvent.args[1])
	if err != nil {
		return EventKill{}, err
	}

	victimSide, err := strconv.Atoi(playerEvent.args[4])
	if err != nil {
		return EventKill{}, err
	}

	return EventKill{
		PlayerEvent:    &playerEvent,
		killerUnitType: playerEvent.args[0],
		killerSide:     killerSide,
		victimPlayerID: playerEvent.args[2],
		victimUnitType: playerEvent.args[3],
		victimSide:     victimSide,
	}, nil
}

func ToFriendlyFireEvent(playerEvent PlayerEvent) (EventFriendlyFire, error) {
	if len(playerEvent.args) != 2 || playerEvent.eventType != FriendlyFire {
		return EventFriendlyFire{}, errors.New(invalidEventMessage + FriendlyFire)
	}

	return EventFriendlyFire{
		PlayerEvent:    &playerEvent,
		weaponName:     playerEvent.args[0],
		victimPlayerID: playerEvent.args[1],
	}, nil
}

func ToPilotDeathEvent(playerEvent PlayerEvent) (EventPilotDeath, error) {
	if len(playerEvent.args) != 1 || playerEvent.eventType != PilotDeath {
		return EventPilotDeath{}, errors.New(invalidEventMessage + PilotDeath)
	}

	return EventPilotDeath{
		PlayerEvent: &playerEvent,
		unitID:      playerEvent.args[0],
	}, nil
}

func ToKilledByEvent(playerEvent PlayerEvent) (EventKilledBy, error) {
	// 1618442167;killed_by;Su-27;1;-1;FA-18C_hornet;2;R-27ET (AA-10 Alamo D)
	if len(playerEvent.args) != 6 || playerEvent.eventType != KilledBy {
		return EventKilledBy{}, errors.New(invalidEventMessage + KilledBy)
	}

	killerSide, err := strconv.Atoi(playerEvent.args[1])
	if err != nil {
		return EventKilledBy{}, err
	}

	victimSide, err := strconv.Atoi(playerEvent.args[4])
	if err != nil {
		return EventKilledBy{}, err
	}

	return EventKilledBy{
		PlayerEvent:    &playerEvent,
		killerUnitType: playerEvent.args[0],
		killerSide:     killerSide,
		killerPlayerID: playerEvent.args[2],
		victimUnitType: playerEvent.args[3],
		victimSide:     victimSide,
		weaponName:     playerEvent.args[5],
	}, nil
}
