package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	mongoUrl = "localhost:27017"
	dbName = "test_db"
	userCollectionName = "user"
)

type ClientManager struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type Client struct {
	id     string
	socket *websocket.Conn
	send   chan []byte
}

type PlayerGone struct {
	Type    string `json:"ty"`
	Players   ListOfPlayersResponse `json:"players"`
}

type Message struct {
	Type    string `json:"ty"`
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}
type Ch struct {
	Name  string `json:"name"`
	Checked bool `json:"checked"`

}


type PlayerName struct {
	Player string `json:"player,omitempty"`
}

const ( //game state
	NotStarted = iota
	WaitingForSuggestion = 1
	SuggestionVoting = 2
	JorneyVoting     = 3
	ShowJorneyResult     = 4
	VictoryForGood     = 5
	VictoryForBad     = 6
)

const (
	JorneyFail    = 1
	JorneySuccess = 2
)

const (
	RegularQuest          = 1
	FlushQuest            = 2
	TwoFailsRequiredQuest = 3
)

type boardConfigurations struct {
	NumOfQuests int
	NumOfBadCharacters int
	PlayersPerLevel []int
	RetriesPerLevel []int
}

var globalConfigPerNumOfPlayers = map[int]boardConfigurations{
	1: {3, 0, []int{1}, []int{5}},
	2: {3, 1, []int{2, 2, 2}, []int{5, 5, 5}},
	4: {4, 1, []int{2, 3, 3, 3}, []int{5, 5, 5, 5}},
	5 : {5, 2, []int{2, 3, 2, 3, 3}, []int{5, 5, 5, 5, 5}},
	6 : {5, 2, []int{2, 3, 4, 3, 4}, []int{5, 5, 5, 5, 5}},
	7 : {7, 3, []int{3, 3, 3, 4, 3, 4, 4}, []int{5, 5, 5, 7, 7, 7, 3}},
	8 : {7, 3, []int{3, 3, 4, 4, 4, 5, 5}, []int{5, 5, 5, 7, 7, 7, 3}},
	9 : {7, 3, []int{3, 4, 4, 5, 4, 5, 5}, []int{5, 5, 5, 7, 7, 7, 3}},
	10  : {7, 4, []int{3, 4, 4, 5, 4, 5, 5}, []int{5, 5, 5, 7, 7, 7, 3}},
	11  : {7, 4, []int{4, 5, 4, 5, 5, 5, 6}, []int{5, 5, 5, 7, 7, 7, 3}},
	12  : {7, 5, []int{4, 5, 5, 6, 5, 6, 6}, []int{5, 5, 5, 7, 7, 7, 3}},
	13   : {8, 5, []int{4, 5, 5, 6, 5, 6, 6, 7}, []int{5, 5, 5, 7, 7, 7, 7, 3}},
}

var goodCh = [...]string {"Puck", "King-Arthur", "Seer", "Titanya", "Galahad", "Nimue", "Sir-Kay", "Good-Angel", "Percival", "Merlin"}

var badCh = [...]string {"Morgana", "Assassin", "Mordred", "Oberon", "Bad-Angel"}

var neutralCharacters = [...]string {"pack"}
var goodCharacters = map[string]bool{
	"Puck": true,
	"King-Arthur": true,
	"Seer" : true,
	"Titanya" : true,
	"Galahad" : true,
	"Nimue": true,
	"Sir-Kay": true,
	"Good-Angel" : true,
	"Percival" : true,
	"Merlin" : true,
}
var badCharacters = map[string]bool{
	"Morgana": true,
	"Assassin": true,
	"Mordred" : true,
	"Oberon" : true,
	"Bad-Angel" : true,
}

type QuestStats struct {
	Final int `json:"final,omitempty"`
	Ppp int `json:"ppp,omitempty"`
	NumOfPlayers int `json:"numofplayers,omitempty"`
	NumOfSuccess int `json:"successes,omitempty"`
	NumOfReversal int `json:"reversals,omitempty"`
	NumOfFailures int `json:"failures,omitempty"`
}

const (
	TITANYA_FIRST_FAIL = iota
)
type QuestManager struct {
	current int //counts from 0
	playersVotes [][]int
	Flags map[int]bool
	results map[int]QuestStats
	realResults map[int]QuestStats
	successfulQuest int
	unsuccessfulQuest int
	playerVotedForCurrent map[string]int
	playerVotedForCurrentQuest []string
	differentResults map[int]int //for king arthur. mapping level to real result
}

type QuestArchiveItem struct {
	PlayersVotedYes []string `json:"playersAcceptedQuest"`
	PlayersVotedNo []string `json:"playersNotAcceptedQuest"`
	Suggester PlayerName `json:"suggester"`
	SuggestedPlayers []string `json:"suggestedPlayers"`
	IsSuggestionAccepted bool `json:"isSuggestionAccepted"`
	IsSuggestionOver bool `json:"isSuggestionOver"`
	NumberOfReversal int `json:"numberOfReversal"`
	NumberOfSuccesses int `json:"numberOfSuccesses"`
	NumberOfFailures int `json:"numberOfFailures"`
	FinalResult int `json:"finalResult"`
	Id float32 `json:"questId"` //e.g. 1.1 , 2 ..
}
type QuestSuggestionsManager struct {
	playersVotedYes []string
	playersVotedNo []string
	unsuccessfulRetries int
	PlayerWithVeto string
	suggesterIndex int
	SuggestedPlayers []string
	SuggestedCharacters map[string]bool
}

type BoardGame struct {

	clientIdToPlayerName map[string]PlayerName

	PlayerNames []PlayerName `json:"players,omitempty"`
	PlayerToCharacter map[PlayerName]string
	CharacterToPlayer map[string]PlayerName
	Characters []string
	quests QuestManager
	archive []QuestArchiveItem
	suggestions QuestSuggestionsManager
	votesForNextMission map[string]bool
	isSuggestionPassed bool
	isSuggestionGood int
	isSuggestionBad int
	manager ClientManager

	QuestStage float32 // e.g. 1, 1.1, 1.2 then 2 ..
	State int `json:"state"`
}

var globalBoard  = BoardGame {
	clientIdToPlayerName: make(map[string]PlayerName),
	QuestStage: 1,
	quests:QuestManager{
		current:               0,
		playersVotes:          make([][]int, 20),
		results:               make(map[int]QuestStats),
		realResults:           make(map[int]QuestStats),
		successfulQuest:       0,
		unsuccessfulQuest:     0,
		playerVotedForCurrent: make(map[string]int),
		playerVotedForCurrentQuest: make([]string, 0),
		differentResults:      make(map[int]int),
		Flags: 				make(map[int]bool),
	},
	archive: make([]QuestArchiveItem, 0),
	manager: ClientManager{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	},
}
var globalMutex sync.RWMutex

func (c *Client) write() {
	defer func() {
		c.socket.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.socket.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (c *Client) read() {
	defer func() {
		globalBoard.manager.unregister <- c
		c.socket.Close()
	}()

	for {
		_, message, err := c.socket.ReadMessage()
		notifyAll := false
		if err != nil {
			fmt.Println("bluppp")

			globalMutex.Lock()
			if globalBoard.State == 0 && len(globalBoard.PlayerNames) > 0 {
				fmt.Println("close from error", globalBoard.PlayerNames)
				index := SliceIndex(len(globalBoard.PlayerNames), func(i int) bool { return globalBoard.PlayerNames[i] == PlayerName{c.id} })
				if index > -1 {
					globalBoard.PlayerNames = removePlayer(globalBoard.PlayerNames, index)
				}
				delete(globalBoard.clientIdToPlayerName, c.id)
			}
			globalMutex.Unlock()

			globalBoard.manager.unregister <- c
			c.socket.Close()
			notifyAll = true
			break
		}
		dd := make(map[string]interface{})
		//fmt.Println(string(message))
		json.Unmarshal(message, &dd)

		//fmt.Println(dd["type"])
		tp := dd["type"]
		isGameCommand := false
		if tp == "add_player" {

			isGameCommand = true
			globalMutex.Lock()
			if globalBoard.State == 0 {
				//fmt.Println(dd["player"])
				player := dd["player"]
				newPlayer := PlayerName{player.(string)}
				globalBoard.clientIdToPlayerName[c.id] = newPlayer
				pls := globalBoard.PlayerNames
				if pls == nil {
					pls = make([]PlayerName, 0)
					globalBoard.PlayerNames = pls
				}

				pls = append(pls, newPlayer)
				globalBoard.PlayerNames = pls
			}
			globalMutex.Unlock()
		} else if tp == "start_game" {
			isGameCommand = true
			var sg StartGameMessage
			json.Unmarshal(message, &sg)
			StartGameHandler(sg.Content)
		} else if tp == "vote_for_suggestion" {
			isGameCommand = true
			var sg VoteForSuggestionMessage
			json.Unmarshal(message, &sg)
			HandleSuggestionVote(sg.Content)
		} else if tp == "suggestion" {
			isGameCommand = true
			var sg SuggestMessage
			json.Unmarshal(message, &sg)
			HandleNewSuggest(sg.Content)
		} else if tp == "vote_for_journey" {
			isGameCommand = true
			var sg VoteForJourneyMessage
			json.Unmarshal(message, &sg)
			HandleJourneyVote(sg.Content)
		} else if tp == "refresh" || notifyAll {
			isGameCommand = true
		} else if tp == "reset" {
			isGameCommand = true
			globalMutex.Lock()
			globalBoard = BoardGame {
				clientIdToPlayerName: globalBoard.clientIdToPlayerName,
				manager: globalBoard.manager,
				PlayerNames:globalBoard.PlayerNames,
				quests: QuestManager{
					current:               0,
					playersVotes:          make([][]int, 20),
					results:               make(map[int]QuestStats),
					realResults:           make(map[int]QuestStats),
					successfulQuest:       0,
					unsuccessfulQuest:     0,
					playerVotedForCurrent: make(map[string]int),
					playerVotedForCurrentQuest: make([]string, 0),
					differentResults:      make(map[int]int),
					Flags: 				make(map[int]bool),
				},
			}

			globalMutex.Unlock()
		}
		if isGameCommand == true {
			for conn := range globalBoard.manager.clients {
				gm := GetGameState(conn.id)
				jsonMessage, _ := json.Marshal(&gm)
				//fmt.Println(string(jsonMessage))
				jsonMessage, _ = json.Marshal(&Message{Sender: c.id, Content: string(jsonMessage)})
			select {
			case conn.send <- jsonMessage:
			default:
			close(conn.send)
			delete(globalBoard.manager.clients, conn)
			}
			}

		} else {
			jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Content: string(message)})
			globalBoard.manager.broadcast <- jsonMessage
		}
	}
}

func (manager *ClientManager) send(message []byte, ignore *Client) {
	for conn := range manager.clients {
		if conn != ignore {
			conn.send <- message
		}
	}
}

func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

func removePlayer(slice []PlayerName, s int) []PlayerName {
	return append(slice[:s], slice[s+1:]...)
}

func getOptionalVotesAccordingToQuestMembers(character string, questMembers map[string]bool, flags map[int]bool, current int, numOfPlayers int) []string {

	if character == "King-Arthur" {
		log.Println("king arthur has fail")
		return []string{"Fail"}
	}
	if character == "Nimue" {
		if _, ok := questMembers["Merlin"]; ok {
			if _, ok := questMembers["Galahad"]; !ok {
				log.Println("nimue  has fail")
				return []string{"Fail"}
			}
		}
	}
	if character == "Titanya" {
		if _, ok := flags[TITANYA_FIRST_FAIL]; !ok {
			log.Println("titanya  has fail")
			return []string{"Fail"}
		}
	}

	if FlushQuest == getTypeOfLevel(current+1, numOfPlayers) {
		if _, ok := badCharacters[character]; ok {
			return []string{"Fail"}
		} else {
			return []string{"Success"}
		}
	}

	res := make([]string, 0)
	if character == "Bad-Angel" || character == "Good-Angel" {
		res = append(res, "Reversal")
	}

	res = append(res, "Success")

	if badCharacters[character] || character == "Puck" {
		res = append(res, "Fail")
	}
	log.Println(character, " has", res)
	return res
}

func (manager *ClientManager) start() {
	for {
		select {
		case conn := <-manager.register:
			if _, ok := manager.clients[conn]; !ok {
				globalMutex.Lock()
				found := false
				for _, v := range globalBoard.PlayerNames {
					if v.Player == conn.id {
						found = true
					}
				}
				if !found {
					globalBoard.PlayerNames = append(globalBoard.PlayerNames, PlayerName{conn.id})
				}

				globalMutex.Unlock()
				manager.clients[conn] = true
				jsonMessage, _ := json.Marshal(&Message{Content: "/A new socket has connected."})
				manager.send(jsonMessage, conn)
		}

		case conn := <-manager.unregister:
			if _, ok := manager.clients[conn]; ok {
				globalMutex.Lock()
				playerName, ok := globalBoard.clientIdToPlayerName[conn.id]
				if ok {
					fmt.Println("close", globalBoard.PlayerNames )
					index := SliceIndex(len(globalBoard.PlayerNames), func(i int) bool { return globalBoard.PlayerNames[i] == playerName })
					globalBoard.PlayerNames = removePlayer(globalBoard.PlayerNames, index)
					fmt.Println("close", index, globalBoard.PlayerNames )
					delete(globalBoard.clientIdToPlayerName, conn.id)
				}

				ls:= ListOfPlayersResponse{Total: len(globalBoard.PlayerNames), Players: globalBoard.PlayerNames}
				playersMsg, _ := json.Marshal(&PlayerGone{Type: "bla", Players:ls})
				//fmt.Println(string(playersMsg))
				globalMutex.Unlock()
				manager.send(playersMsg, conn)

				close(conn.send)
				delete(manager.clients, conn)
				jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected."})
				manager.send(jsonMessage, conn)
			}
		case message := <-manager.broadcast:
			for conn := range manager.clients {
				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(manager.clients, conn)
				}
			}
		}
	}
}


type StartGameMessage struct {
	Tp string `json:"type"`
	Content []Ch `json:"content"`

}

type VoteForSuggestionMessage struct {
	Tp string `json:"type"`
	Content VoteForSuggestion `json:"content"`

}

type VoteForSuggestion struct {
	PlayerName string `json:"playerName"`
	Vote bool `json:"vote"`

}

type SuggestMessage struct {
	Tp string `json:"type"`
	Content ListOfSuggestions `json:"content"`

}

type VoteForJourneyMessage struct {
	Tp string `json:"type"`
	Content VoteForJourney `json:"content"`

}

func StartGameHandler(newGameConfig []Ch) {
	fmt.Println("newGameConfig", newGameConfig)
	globalMutex.Lock()

	chosenCharacters := make([]string, 0)
	numOfPlayers := len(globalBoard.PlayerNames)
	requiredBads := globalConfigPerNumOfPlayers[numOfPlayers].NumOfBadCharacters

	var numOfBads int
	var numOfGood int
	for _, v := range newGameConfig {
		if v.Checked == true {
			if badCharacters[v.Name] == true {
				numOfBads++
			} else if goodCharacters[v.Name] == true {
				numOfGood++
			}

		}
	}

	//sanity
	if requiredBads != numOfBads {
		globalMutex.Unlock()
		return
	}

	//sanity
	if numOfPlayers != (numOfGood + numOfBads) {
		globalMutex.Unlock()
		return
	}

	chosenCharacters = assignCharactersToRegisteredPlayers(newGameConfig, chosenCharacters)

	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(chosenCharacters), func(i, j int) {
		chosenCharacters[i], chosenCharacters[j] = chosenCharacters[j], chosenCharacters[i]
	})
	fmt.Println("ttttttttttttttttt", chosenCharacters)
	globalBoard.State = WaitingForSuggestion
	if globalBoard.quests.results == nil {
		globalBoard.quests.results = make(map[int]QuestStats)
	}

	globalBoard.Characters = chosenCharacters
	for i:=0; i < globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests; i++ {
		en := QuestStats{}
		fmt.Println(i)
		en.Ppp = getTypeOfLevel(i+1, len(globalBoard.PlayerNames))
		en.NumOfPlayers = globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].PlayersPerLevel[i]
		globalBoard.quests.results[i+1] = en
		fmt.Println(en)
	}
	globalBoard.suggestions.suggesterIndex = 0

	numOfUnsuccesfulRetries:= globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].RetriesPerLevel[globalBoard.quests.current]
	suggesterVetoIn := (globalBoard.suggestions.suggesterIndex+numOfUnsuccesfulRetries-1) % len(globalBoard.PlayerNames)
	globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[suggesterVetoIn].Player

	globalMutex.Unlock()
}

type SecretResponse struct {
	Character string `json:"character,omitempty"`
	Secrets []string `json:"secrets,omitempty"`
}

func GetNightSecretsFromPlayerName(player PlayerName) SecretResponse {
	response := SecretResponse{}

	if player.Player == "" {
		return SecretResponse{}
	}

	character := globalBoard.PlayerToCharacter[player]
	response.Character = character
	if character == "Merlin" {
		for k, v := range globalBoard.CharacterToPlayer {
			fmt.Println(k, "and ", badCharacters[k])
			if _, ok := badCharacters[k]; ok && k != "Mordred" {
				response.Secrets = append(response.Secrets, v.Player + " is bad")
			}
			if k == "Sir-Kay" {
				response.Secrets = append(response.Secrets, v.Player + " is bad")
			}
		}
	}
	if character == "Nimue" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Galahad" {
				response.Secrets = append(response.Secrets, v.Player + " is Galahad")
			}
			if k == "Merlin" {
				response.Secrets = append(response.Secrets, v.Player + " is Merlin")
			}
		}
	}
	if character == "Percival" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Morgana" {
				response.Secrets = append(response.Secrets, v.Player + " is Morgana/Merlin")
			}
			if k == "Merlin" {
				response.Secrets = append(response.Secrets, v.Player + " is Morgana/Merlin")
			}
		}
	}
	if _, ok := badCharacters[character] ; ok && character != "Oberon" {
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; ok && k != character && k != "Oberon" {
				response.Secrets = append(response.Secrets, v.Player+" is bad")
			}
		}
	}
	return response
}

func assignCharactersToRegisteredPlayers(newGameConfig []Ch, chosenCharacters []string) []string {
	for _, v := range newGameConfig {
		if v.Checked == true {
			chosenCharacters = append(chosenCharacters, v.Name)
		}
	}

	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(chosenCharacters), func(i, j int) {
		chosenCharacters[i], chosenCharacters[j] = chosenCharacters[j], chosenCharacters[i]
	})

	globalBoard.PlayerToCharacter = make(map[PlayerName]string)
	globalBoard.CharacterToPlayer = make(map[string]PlayerName)

	for i := 0; i < len(globalBoard.PlayerNames); i++ {
		globalBoard.PlayerToCharacter[globalBoard.PlayerNames[i]] = chosenCharacters[i]
		globalBoard.CharacterToPlayer[chosenCharacters[i]] = globalBoard.PlayerNames[i]
	}
	return chosenCharacters
}

type ListOfPlayersResponse struct {
	Total int `json:"total,omitempty"`
	Players []PlayerName `json:"all,omitempty"`
}

func getTypeOfLevel(levelNum int, numOfPlayers int) int {
	fmt.Println(levelNum, numOfPlayers)
	if numOfPlayers <= 6 {
		return RegularQuest
	}
	if levelNum == 4 {
		return FlushQuest
	} else if globalConfigPerNumOfPlayers[numOfPlayers].NumOfQuests - 1 == levelNum {
		return TwoFailsRequiredQuest
	}
	return RegularQuest

}

type ListOfSuggestions struct {
	Total int `json:"total,omitempty"`
	Players []PlayerName2 `json:"all,omitempty"`
}

type PlayerName2 struct {
	Player string `json:"player,omitempty"`
	Ch bool `json:"ch,omitempty"`
}

func HandleNewSuggest(pl ListOfSuggestions) {
	globalMutex.Lock()
	suggestedPlayers := make([]string, 0)
	suggestedCharacters := make(map[string]bool, 0)

	for _, v := range pl.Players {
		if v.Ch == true {
			suggestedPlayers= append(suggestedPlayers, v.Player)
			suggestedCharacters[globalBoard.PlayerToCharacter[PlayerName{v.Player}]] = true
		}
	}

	suggesterIn := globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	newEntry := QuestArchiveItem{Id:globalBoard.QuestStage, Suggester:globalBoard.PlayerNames[suggesterIn], SuggestedPlayers:suggestedPlayers}

	log.Println("SuggestedPlayers:", suggestedPlayers)
	globalBoard.suggestions.SuggestedPlayers = suggestedPlayers
	globalBoard.suggestions.SuggestedCharacters = suggestedCharacters
	globalBoard.State = SuggestionVoting
	globalBoard.votesForNextMission = make(map[string]bool)
	globalBoard.suggestions.playersVotedYes = make([]string, 0)
	globalBoard.suggestions.playersVotedNo = make([]string, 0)

	if globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].RetriesPerLevel[globalBoard.quests.current]-1 == globalBoard.suggestions.unsuccessfulRetries {
		globalBoard.State = JorneyVoting
		globalBoard.suggestions.unsuccessfulRetries = 0

		allPlayers := make([]string, len(globalBoard.PlayerNames))
		for _, player := range globalBoard.PlayerNames {
			allPlayers = append(allPlayers, player.Player)
		}
		newEntry.IsSuggestionAccepted = true
		newEntry.IsSuggestionOver = true
		newEntry.PlayersVotedYes = allPlayers
		globalBoard.suggestions.playersVotedYes = allPlayers
		globalBoard.QuestStage += 0.01
		globalBoard.QuestStage = float32(math.Ceil(float64(globalBoard.QuestStage)))
		globalBoard.isSuggestionGood, globalBoard.isSuggestionBad = 0, 0
		globalBoard.suggestions.suggesterIndex++

	}
	globalBoard.archive = append(globalBoard.archive, newEntry)
	globalMutex.Unlock()

}


type GameState struct {
	Players ListOfPlayersResponse `json:"players,omitempty"`
	CurrentQuest int `json:"current,omitempty"`
	Characters []string `json:"characters,omitempty"`
	Size int `json:"size,omitempty"`
	State int `json:"state"`
	Archive []QuestArchiveItem `json:"archive"`
	Secrets SecretResponse `json:"secrets"`
	Suggester string `json:"suggester,omitempty"`
	OptionalVotes []string `json:"optionalVotes,omitempty"`
	SuggesterVeto string `json:"suggesterVeto,omitempty"`
	SuggestedPlayers []string `json:"suggestedPlayers,omitempty"`
	PlayersVotedForCurrQuest []string `json:"PlayersVotedForCurrQuest,omitempty"`
	PlayersVotedYes []string `json:"PlayersVotedYesForSuggestion,omitempty"`
	PlayersVotedNo []string `json:"PlayersVotedNoForSuggestion,omitempty"`
	Results map[int]QuestStats `json:"results,omitempty"`
}

func GetGameState(clientId string) GameState  {
	globalMutex.RLock()
	board := GameState{}

	board.Players.Total = len(globalBoard.PlayerNames)
	board.Players.Players = globalBoard.PlayerNames
	board.SuggestedPlayers = globalBoard.suggestions.SuggestedPlayers
	board.CurrentQuest = globalBoard.quests.current + 1
	board.CurrentQuest = globalBoard.quests.current + 1
	board.Size = globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests
	board.Results = globalBoard.quests.results
	board.Characters = globalBoard.Characters
	board.PlayersVotedForCurrQuest = globalBoard.quests.playerVotedForCurrentQuest
	board.SuggesterVeto = globalBoard.suggestions.PlayerWithVeto
	cpy := make([]QuestArchiveItem, len(globalBoard.archive))
	copy(cpy, globalBoard.archive)
	if len(cpy) > 0 && globalBoard.State == SuggestionVoting {
		cpy[len(cpy)-1].PlayersVotedYes = make([]string, 0)
		cpy[len(cpy)-1].PlayersVotedNo = make([]string, 0)
	}
	if len(cpy) > 0 && globalBoard.State == JorneyVoting {
		cpy[len(cpy)-1].NumberOfReversal = 0
		cpy[len(cpy)-1].NumberOfSuccesses = 0
		cpy[len(cpy)-1].NumberOfFailures = 0
	}
	board.Archive = cpy

	board.State = globalBoard.State
	board.Secrets = GetNightSecretsFromPlayerName(PlayerName{clientId})
	board.OptionalVotes = getOptionalVotesAccordingToQuestMembers(globalBoard.PlayerToCharacter[PlayerName{clientId}], globalBoard.suggestions.SuggestedCharacters, globalBoard.quests.Flags, globalBoard.quests.current, len(globalBoard.PlayerNames))
	board.PlayersVotedYes = globalBoard.suggestions.playersVotedYes
	board.PlayersVotedNo = globalBoard.suggestions.playersVotedNo
	if len(globalBoard.PlayerNames) > 0 {
		board.Suggester = globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player
	}

	globalMutex.RUnlock()
	return board
}


type VoteForJourney struct {
	PlayerName string `json:"playerName,omitempty"`
	Vote int `json:"vote,omitempty"`
}


func HandleJourneyVote(vote VoteForJourney) {
	globalMutex.Lock()
	current := globalBoard.quests.current

	if globalBoard.State != JorneyVoting  {
		return
	}

	if _, ok := globalBoard.quests.playerVotedForCurrent[vote.PlayerName]; ok {
		globalMutex.Unlock()
		return
	}

	if globalBoard.PlayerToCharacter[PlayerName{vote.PlayerName}] == "Titanya" && vote.Vote==0 {
		globalBoard.quests.Flags[TITANYA_FIRST_FAIL] = true
	}

	log.Println(vote.PlayerName, " voted ", vote.Vote)
	globalBoard.quests.playerVotedForCurrentQuest = append(globalBoard.quests.playerVotedForCurrentQuest, vote.PlayerName)
	globalBoard.quests.playerVotedForCurrent[vote.PlayerName] = vote.Vote
	mp := append(globalBoard.quests.playersVotes[current], vote.Vote)
	requiredVotes := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].PlayersPerLevel[current]

	res := globalBoard.quests.results[current+1]

	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	if vote.Vote == 0 {
		res.NumOfFailures++
		curEntry.NumberOfFailures++
	} else if vote.Vote == 1 {
		res.NumOfSuccess++
		curEntry.NumberOfSuccesses++
	} else {
		res.NumOfReversal++
		curEntry.NumberOfReversal++
	}

	if len(mp) == requiredVotes { //last vote
		retriesPerLevel := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].RetriesPerLevel
		if globalBoard.quests.current+1 < len(retriesPerLevel) { //not last quest in game
			numOfUnsuccesfulRetries := retriesPerLevel[globalBoard.quests.current+1]
			suggesterVetoIn := (globalBoard.suggestions.suggesterIndex+numOfUnsuccesfulRetries-1) % len(globalBoard.PlayerNames)
			globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[suggesterVetoIn].Player
		}

		res.Final = CalculateQuestResult(mp)
		log.Println("Quest Result:(", globalBoard.quests.current+1, ")", res.Final)
		curEntry.FinalResult = res.Final
		globalBoard.quests.results[current+1] = res

		if globalBoard.quests.results[current+1].Final == JorneySuccess {
			globalBoard.quests.successfulQuest++
		} else {
			globalBoard.quests.unsuccessfulQuest++
		}

		if playerName, ok := globalBoard.CharacterToPlayer["King-Arthur"]; ok {
			//King-Arthur is playing
			if vote, ok := globalBoard.quests.playerVotedForCurrent[playerName.Player]; ok {
				//King-Arthur was in this quest
				if vote==0 {
					// King-Arthur voted "Fail"
					log.Println("switch King-Arthur's \"Fail\" to \"Success")
					realResults := make([]int, len(mp))
					copy(realResults, mp)
					for i, res := range realResults {
						if res == 0 {
							realResults[i] = 1
							break
						}
					}
					realFinal := CalculateQuestResult(realResults)
					log.Println("Original quest result: ", res.Final, "actual quest result: ", realFinal)
					if res.Final != realFinal {
						globalBoard.quests.differentResults[current+1] = realFinal
						if res.Final == JorneySuccess {
							globalBoard.quests.successfulQuest--
							globalBoard.quests.unsuccessfulQuest++
						} else {
							globalBoard.quests.successfulQuest++
							globalBoard.quests.unsuccessfulQuest--
						}
					}
				}
			}
		}

		numOfExpectedQuests := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests
		if globalBoard.quests.successfulQuest > numOfExpectedQuests/ 2 {
			globalBoard.State = VictoryForGood
		} else if globalBoard.quests.unsuccessfulQuest > numOfExpectedQuests/ 2 {
			globalBoard.State = VictoryForBad
		} else {
			globalBoard.State = ShowJorneyResult
			globalBoard.votesForNextMission = make(map[string]bool)
		}

		globalBoard.quests.playerVotedForCurrentQuest = make([]string, 0)
		globalBoard.quests.playerVotedForCurrent = make(map[string]int)
		globalBoard.votesForNextMission = make(map[string]bool)
		globalBoard.suggestions.SuggestedPlayers = make([]string, 0)

	}
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry
	globalBoard.quests.results[current+1] = res
	globalBoard.quests.playersVotes[current] = mp

	if len(mp) == requiredVotes { //last vote
		globalBoard.quests.current++
	}

	globalMutex.Unlock()
}

func CalculateQuestResult(mp []int) int {
	result := JorneySuccess
	fmt.Println("++ last")
	NumOfFailures := 0
	NumOfReverse := 0
	for _, v := range mp {
		if v == 0 {
			NumOfFailures++
		}
		if v == 2 {
			NumOfReverse++
		}
	}

	questType := getTypeOfLevel(globalBoard.quests.current+1, len(globalBoard.PlayerNames))
	if questType == FlushQuest {
		if NumOfFailures == 1 {
			result = JorneyFail
		}
	} else if questType == TwoFailsRequiredQuest {

		if NumOfFailures >= 2 {
			result = JorneyFail
		}
		if NumOfReverse >= 1 {
			if NumOfFailures == 0 {
				result = JorneyFail
			}
			if NumOfFailures == 1 {
				result = JorneyFail
			}
			if NumOfFailures >= 2 {
				result = JorneySuccess
			}
			NumOfReverse--
		}
	} else {
		if NumOfFailures > 0 { //regular case
			result = JorneyFail
		}
	}
	if NumOfReverse > 0 {
		if NumOfReverse%2 != 0 {
			if result == JorneyFail {
				result = JorneySuccess
			} else {
				result = JorneyFail
			}
		}
	}
	return result
}

func HandleSuggestionVote(vote VoteForSuggestion) {
	log.Println("suggestion -  ", vote.PlayerName, " voted ", vote.Vote)

	globalMutex.Lock()
	if globalBoard.votesForNextMission == nil {
		globalBoard.votesForNextMission = make(map[string]bool)
	}

	if _, ok := globalBoard.votesForNextMission[vote.PlayerName]; ok {
		globalMutex.Unlock()
		return
	}

	globalBoard.votesForNextMission[vote.PlayerName] = vote.Vote
	curEntry := globalBoard.archive[len(globalBoard.archive)-1]
	if vote.Vote == true {
		globalBoard.suggestions.playersVotedYes = append(globalBoard.suggestions.playersVotedYes, vote.PlayerName)
		curEntry.PlayersVotedYes = append(curEntry.PlayersVotedYes, vote.PlayerName)
		globalBoard.isSuggestionGood++ //inc good counter
	} else {
		globalBoard.suggestions.playersVotedNo = append(globalBoard.suggestions.playersVotedNo, vote.PlayerName)
		curEntry.PlayersVotedNo = append(curEntry.PlayersVotedNo, vote.PlayerName)
		globalBoard.isSuggestionBad++ //inc bad counter
	}

	if len(globalBoard.votesForNextMission) == len(globalBoard.PlayerNames) { //last vote
		curEntry.IsSuggestionOver = true
		if globalBoard.isSuggestionGood > globalBoard.isSuggestionBad {
			globalBoard.State = JorneyVoting
			globalBoard.suggestions.unsuccessfulRetries = 0
			curEntry.IsSuggestionAccepted = true
			globalBoard.QuestStage += 0.01
			globalBoard.QuestStage = float32(math.Ceil(float64(globalBoard.QuestStage)))

			numOfUnsuccesfulRetries:= globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].RetriesPerLevel[globalBoard.quests.current]
			suggesterVetoIn := (globalBoard.suggestions.suggesterIndex+1+numOfUnsuccesfulRetries) % len(globalBoard.PlayerNames)
			globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[suggesterVetoIn].Player
		} else {
			globalBoard.State = WaitingForSuggestion
			globalBoard.QuestStage += 0.1
			globalBoard.QuestStage = float32(math.Round(float64(globalBoard.QuestStage*100)) / 100)
			globalBoard.suggestions.unsuccessfulRetries++
		}

		globalBoard.isSuggestionGood, globalBoard.isSuggestionBad = 0, 0
		globalBoard.suggestions.suggesterIndex++
	}
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry

	globalMutex.Unlock()
}


type ListAllVotesResponse struct {
	IsPassed bool `json:"isPassed,omitempty"`
	ErrorMsg string `json:"errorCode,omitempty"`
	VotePerPlayer map[string]bool `json:"allVotes,omitempty"`
}


func wsPage(res http.ResponseWriter, req *http.Request) {
	jwtToken := req.URL.Query().Get("token")
	log.Println(jwtToken)

	claims, err := jwt.ParseWithClaims(jwtToken, &JWTData{}, func(token *jwt.Token) (interface{}, error) {
		if jwt.SigningMethodHS256 != token.Method {
			return nil, errors.New("Invalid signing algorithm")
		}
		return []byte(SECRET), nil
	})

	if err != nil {
		log.Println(err)
		http.Error(res, "Request failed!", http.StatusUnauthorized)
	}

	data := claims.Claims.(*JWTData)

	userName := data.CustomClaims["userName"]

	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
	if error != nil {
		http.NotFound(res, req)
		return
	}



	//uuid,_:= uuid.NewV4()
	client := &Client{id: userName, socket: conn, send: make(chan []byte)}

	globalBoard.manager.register <- client

	go client.read()
	go client.write()

}

type userRouter struct {
	userService *UserService1
}

func main() {
	fmt.Println("Starting server at http://localhost:12345...")
	session, _ := NewSession(mongoUrl)
	defer func() {
		session.Close()
	}()

	hash := Hash{}
	userService := NewUserService(session.Copy(), dbName, userCollectionName, &hash)
	userRouter := userRouter{userService}

	go globalBoard.manager.start()
	router := mux.NewRouter()
	router.HandleFunc("/ws", wsPage).Methods("GET")
	router.HandleFunc("/register", userRouter.createUserHandler).Methods("PUT", "OPTIONS", "POST")
	router.HandleFunc("/login", userRouter.login).Methods("POST", "OPTIONS")
	log.Fatal(http.ListenAndServe(":12345", cors.AllowAll().Handler(router)))
}

type user struct {
	Email string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
}

type JWT struct {
	Token string `json:"token,omitempty"`
}
var dbUsers = map[string]user{}      // user ID, user
var dbSessions = map[string]string{} // session ID, user ID


func (ur *userRouter) createUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	user, err := decodeUser(r)

	if err != nil {
		fmt.Println("error", err)
		return
	}
fmt.Println(user)
	err = ur.userService.Create(&user)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("good")
}


const (
	PORT   = "1337"
	SECRET = "42isTheAnswer"
)

func (ur *userRouter) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	var user struct {
		User string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&user)
	dbUser, err := ur.userService.GetByUsername(user.User)
	if err != nil {

		fmt.Println(err)
		return
	}
	c := Hash{}
	fmt.Println("login start")
	compareError := c.Compare(dbUser.Password, user.Password)
	if compareError == nil {
		claims := JWTData{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour*7).Unix(),
			},

			CustomClaims: map[string]string{
				"userName": dbUser.Username,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(SECRET))
		if err != nil {
			log.Println(err)
			http.Error(w, "Login failed!", http.StatusUnauthorized)
		}

		json, err := json.Marshal(struct {
			Token string `json:"token"`
			Name string `json:"name"`
		}{
			tokenString,
			dbUser.Username,
		})

		if err != nil {
			log.Println(err)
			http.Error(w, "Login failed!", http.StatusUnauthorized)
		}

		w.Write(json)
	} else {
		http.Error(w, "Login failed!", http.StatusUnauthorized)
	}
	fmt.Println("login end")
}

func decodeUser(r *http.Request) (User, error) {
	var u User
	if r.Body == nil {
		return u, errors.New("no request body")
	}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&u)
	return u, err
}

type JWTData struct {
	// Standard claims are the standard jwt claims from the IETF standard
	// https://tools.ietf.org/html/rfc7519
	jwt.StandardClaims
	CustomClaims map[string]string `json:"custom,omitempty"`
}

