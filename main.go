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
	"os"
	"strconv"
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

type GameConfiguration struct{
	Characters []Ch `json:"characters"`
	Excalibur bool `json:"excalibur"`
}

type Ch struct {
	Name  string `json:"name"`
	Checked bool `json:"checked"`
	Assassin bool  `json:"assassin"`
}

type PlayerNameWithCharacter struct {
	Player string `json:"player,omitempty"`
	Character string `json:"character,omitempty"`
}

type PlayerName struct {
	Player string `json:"player,omitempty"`
}

const ( //game state
	NotStarted = iota
	SirPickPlayer = 1
	WaitingForSuggestion = 2
	SuggestionVoting = 3
	JorneyVoting     = 4
	ExcaliburPick     = 5
	VictoryForGood     = 6
	VictoryForBad     = 7
	MurdersAfterGoodVictory     = 8
	MurdersAfterBadVictory     = 9
	VictoryForGawain    = 10
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

var neutralCharacters = map[string]bool{
	"Ginerva": true,
	"Puck": true,
	"Gawain": true,
}


var goodCharacters = map[string]bool{
	"Viviana": true,
	"King-Arthur": true,
	"Seer" : true,
	"Titanya" : true,
	"Galahad" : true,
	"Nimue": true,
	"Sir-Kay": true,
	"Good-Angel" : true,
	"Percival" : true,
	"Merlin" : true,
	"Tristan" : true,
	"Iseult" : true,
	"Prince-Claudin" : true,
	"Nirlem" : true,
	"Sir-Robin" : true,
	"Pellinore": true,
	"Lot": true,
	"Cordana": true,
	"The-Coward": true,
	"Merlin-Apprentice": true,
	"Lancelot-Good": true,
	"Guinevere": true,
	"Galaad":true,
	"Balain" :true,
}
var badCharacters = map[string]bool{
	"Morgana": true,
	"Assassin": true,
	"Mordred" : true,
	"Oberon" : true,
	"Bad-Angel" : true,
	"King-Claudin": true,
	"Polygraph": true,
	"The-Questing-Beast": true,
	"Accolon": true,
	"Lancelot-Bad": true,
	"Queen-Mab": true,
	"Balin": true,
}

type QuestStats struct {
	Final int `json:"final,omitempty"`
	Ppp int `json:"ppp,omitempty"`
	NumOfPlayers int `json:"numofplayers,omitempty"`
	NumOfSuccess int `json:"successes,omitempty"`
	NumOfReversal int `json:"reversals,omitempty"`
	NumOfFailures int `json:"failures,omitempty"`
	NumOfBeasts int `json:"beasts,omitempty"`
}

const (
	TITANYA_FIRST_FAIL = iota
	BEAST_FIRST_SUCCESS
	HAS_TWO_LANCELOT
	HAS_ONLY_GOOD_LANCELOT
	HAS_ONLY_BAD_LANCELOT
	EXCALIBUR
)

type ExcaliburVote struct {
	Player string  `json:"player"`
	Vote int   `json:"vote"`
}

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
	IsSwitchLancelot bool `json:"switch"`
	NumberOfReversal int `json:"numberOfReversal"`
	NumberOfSuccesses int `json:"numberOfSuccesses"`
	NumberOfFailures int `json:"numberOfFailures"`
	NumberOfBeasts int `json:"numberOfBeasts"`
	FinalResult int `json:"finalResult"`
	Id float32 `json:"questId"` //e.g. 1.1 , 2 ..
	ExcaliburPlayer string  `json:"excaliburPicker"`
	ExcaliburChosenPlayer string  `json:"excaliburChoose"`
}
type QuestSuggestionsManager struct {
	playersVotedYes []string
	playersVotedNo []string
	unsuccessfulRetries int
	PlayerWithVeto string
	suggesterIndex int
	SuggestedPlayers []string
	SuggestedTemporaryPlayers string //showed until picking all quest memebers
	SuggestedCharacters map[string]bool
	excalibur Excalibur
}

type ExcaliburPickResponse struct {
	player string
}

const (
	ALL_GOOD = "Good"
	ALL_BAD = "Bad"
)
type Murder struct {
	target          []string
	TargetCharacters  []string `json:"target"`
	By              string `json:"by"`
	ByCharacter    string `json:"byCharacter"`
	stopIfSucceeded bool
	StateAfterSuccess	int
}

type MurderInfo struct {
	by              []string
}

type MurderResult struct {
	target          []string
	targetCharacter  []string
	by              string
	byCharacter    string
	success    bool
}

type MurderItem struct{
	targetPlayer string `json:"target"`
	byPlayer string `json:"by"`
}



type BoardGame struct {

	clientIdToPlayerName map[string]PlayerName

	playersWithGoodCharacter []string //for vivian
	playersWithBadCharacter []string //for vivian
	Secrets map[string][]string
	PlayerNames []PlayerName `json:"players,omitempty"`
	PlayerToCharacter map[PlayerName]string
	CharacterToPlayer map[string]PlayerName
	Characters []string
	PendingMurders []Murder
	PlayerToMurderInfo map[string]MurderInfo
	quests QuestManager
	archive []QuestArchiveItem
	lancelotCards []int
	lancelotCardsIndex int
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
	playersWithBadCharacter: make([]string, 0),
	playersWithGoodCharacter: make([]string, 0),
	clientIdToPlayerName: make(map[string]PlayerName),
	QuestStage: 1,
	lancelotCards: make([]int, 7),
	Secrets: make(map[string][]string),
	PlayerToMurderInfo: make(map[string]MurderInfo),
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
		//globalBoard.manager.unregister <- c
		log.Println("defer")
		//c.socket.Close()
	}()
	log.Println("client read start")
	for {
		_, message, err := c.socket.ReadMessage()
		notifyAll := false
		if err != nil {
			log.Println("bluppp")

			globalMutex.Lock()
			if (globalBoard.State == 0 || globalBoard.State > 5) && len(globalBoard.PlayerNames) > 0 {
				log.Println("close from error", globalBoard.PlayerNames)
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
		//log.Println(string(message))
		json.Unmarshal(message, &dd)

		var isOnlyForAllExceptSender bool
		isOnlyForSender := false
		//log.Println(dd["type"])
		tp := dd["type"]
		isGameCommand := false
		if tp == "add_player" {

			isGameCommand = true
			globalMutex.Lock()
			if globalBoard.State == 0 {
				//log.Println(dd["player"])
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
		} else if tp == "murder" {
			isGameCommand = true
			var sg MurderMessage
			json.Unmarshal(message, &sg)
			HandleMurder(sg.Content)
		} else if tp == "sir_pick" {
			isGameCommand = true
			var sg SirMessage
			json.Unmarshal(message, &sg)
			HandleSir(sg.Content)
		} else if tp == "excalibur_pick" {
			isGameCommand = true
			var sg ExcaliburMessage
			json.Unmarshal(message, &sg)
			ExcaliburHandler(sg.Content)
		} else if tp == "vote_for_suggestion" {
			isGameCommand = true
			var sg VoteForSuggestionMessage
			json.Unmarshal(message, &sg)
			log.Println("=====================>")
			HandleSuggestionVote(sg.Content)
			log.Println("<=====================")
		} else if tp == "suggestion" {
			isGameCommand = true
			var sg SuggestMessage
			json.Unmarshal(message, &sg)
			HandleNewSuggest(sg.Content)
		} else if tp == "suggestion_tmp" {
			isOnlyForAllExceptSender = true
			isGameCommand = true
			var sg SuggestTmpMessage
			json.Unmarshal(message, &sg)
			HandleTemporarySuggest(sg.Content)
		} else if tp == "vote_for_journey" {
			isGameCommand = true
			var sg VoteForJourneyMessage
			json.Unmarshal(message, &sg)
			HandleJourneyVote(sg.Content)
		} else if tp == "refresh" || notifyAll {
			if globalBoard.State == 0 {
				isOnlyForSender = false
			} else {
				isOnlyForSender = true
			}
			isGameCommand = true
		} else if tp == "reset" {
			isGameCommand = true
			globalMutex.Lock()
			globalBoard = BoardGame {
				QuestStage: 1,
				lancelotCards: make([]int, 7),
				playersWithBadCharacter: make([]string, 0),
				playersWithGoodCharacter: make([]string, 0),
				Secrets: make(map[string][]string),
				clientIdToPlayerName: globalBoard.clientIdToPlayerName,
				manager: globalBoard.manager,
				PlayerToMurderInfo: make(map[string]MurderInfo),
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
			recipient := ""
			if isOnlyForSender {
				recipient = c.id
			}
			if isOnlyForAllExceptSender {
				recipient = "^" + c.id
			}
			jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Recipient:recipient, Content:"board"})
			globalBoard.manager.broadcast <- jsonMessage
		} else {
			jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Content: string(message)})
			globalBoard.manager.broadcast <- jsonMessage
		}

		if notifyAll {
			break
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

	if character == "Gawain" {
		return []string{"Fail", "Success"}
	}

	if character == "King-Arthur" {
		return []string{"Fail"}
	}
	if character == "Lancelot-Bad" {
		return []string{"Fail"}
	}
	if character == "Titanya" {
		numOfExpectedQuests := globalConfigPerNumOfPlayers[numOfPlayers].NumOfQuests
		if globalBoard.quests.unsuccessfulQuest+1 > numOfExpectedQuests/ 2 {
			return []string{"Success"}
		}
		if _, ok := flags[TITANYA_FIRST_FAIL]; !ok {
			log.Println("titanya  has fail")
			return []string{"Fail"}
		}
	}

	if character == "Polygraph" {
		return []string{"Fail"}
	}
	if character == "Lot" {
		return []string{"Success"}
	}
	if character == "Nimue" {
		if _, ok := questMembers["Merlin"]; ok {
			if _, ok := questMembers["Galahad"]; !ok {
				log.Println("nimue  has fail")
				return []string{"Fail"}
			}
		}
	}

	if FlushQuest == getTypeOfLevel(current+1, numOfPlayers) {
		if _, ok := badCharacters[character]; ok || character == "Ginerva" {
			return []string{"Fail"}
		} else {
			return []string{"Success"}
		}
	}

	if character == "The-Questing-Beast" {
		if _, ok := flags[BEAST_FIRST_SUCCESS]; !ok {
			return []string{"Success", "Beast"}
		} else {
			return []string{"Beast"}
		}
	}

	res := make([]string, 0)
	if character == "Bad-Angel" || character == "Good-Angel" {
		res = append(res, "Reversal")
	}

	res = append(res, "Success")

	if badCharacters[character] || character == "Puck" || character == "Ginerva" {
		res = append(res, "Fail")
	}
	log.Println(character, " has", res)
	return res
}

func (manager *ClientManager) start() {
	for {
		log.Println("con register")
		select {
		case conn := <-manager.register:
			log.Println("registerss")
			if _, ok := manager.clients[conn]; !ok {
				globalMutex.Lock()
				found := false
				for _, v := range globalBoard.PlayerNames {
					if v.Player == conn.id {
						found = true
					}
				}
				if !found && (globalBoard.State == NotStarted || globalBoard.State > 5) {
					log.Println("Add", conn.id)
					globalBoard.PlayerNames = append(globalBoard.PlayerNames, PlayerName{conn.id})
				}

				globalMutex.Unlock()
				manager.clients[conn] = true
				jsonMessage, _ := json.Marshal(&Message{Content: "/A new socket has connected."})
				manager.send(jsonMessage, conn)
			}
		case conn := <-manager.unregister:
			log.Println("con unregister")
			if _, ok := manager.clients[conn]; ok {
				globalMutex.Lock()
				//playerName, ok := globalBoard.clientIdToPlayerName[conn.id]
				if ok {
					log.Println("close", globalBoard.PlayerNames )
					if globalBoard.State == NotStarted || globalBoard.State > 5 {
						index := SliceIndex(len(globalBoard.PlayerNames), func(i int) bool { return globalBoard.PlayerNames[i] == PlayerName{conn.id} })
						if index >= 0 {
							globalBoard.PlayerNames = removePlayer(globalBoard.PlayerNames, index)
							log.Println("close", index, globalBoard.PlayerNames)
						}
					}

					delete(globalBoard.clientIdToPlayerName, conn.id)
				}

				ls:= ListOfPlayersResponse{Total: len(globalBoard.PlayerNames), Players: globalBoard.PlayerNames}
				playersMsg, _ := json.Marshal(&PlayerGone{Type: "bla", Players:ls})
				//log.Println(string(playersMsg))
				globalMutex.Unlock()
				manager.send(playersMsg, conn)

				close(conn.send)
				delete(manager.clients, conn)
				jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected."})
				manager.send(jsonMessage, conn)
			}
		case message := <-manager.broadcast:
			log.Println("con broadcast")
			var msg Message
			json.Unmarshal(message, &msg)
			for conn := range manager.clients {
				if msg.Content == "board" {
					if msg.Recipient != "" && msg.Recipient[0] != '^' && msg.Recipient != conn.id {
						continue
					}
					if msg.Recipient != "" && msg.Recipient[0] == '^' && msg.Recipient[1:] == conn.id {
						continue
					}
					gm := GetGameState(conn.id)
					jsonMessage, _ := json.Marshal(&gm)
					if globalBoard.PlayerToCharacter[PlayerName{conn.id}] == "Viviana" {
						log.Println("VIVIANA")
						log.Println(string(jsonMessage))
					}
					log.Println(string(jsonMessage))
					message, _ = json.Marshal(&Message{Sender: msg.Sender, Content: string(jsonMessage)})
				}


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
	Content GameConfiguration `json:"content"`

}

type ExcaliburMessage struct {
	Tp string `json:"type"`
	Content []string `json:"content"`
}

type VoteForSuggestionMessage struct {
	Tp string `json:"type"`
	Content VoteForSuggestion `json:"content"`

}

type MurderMsg struct {
	Selection []PlayerNameMurder `json:"all"`

}



type VoteForSuggestion struct {
	PlayerName string `json:"playerName"`
	Vote bool `json:"vote"`

}

type SuggestMessage struct {
	Tp string `json:"type"`
	Content Suggestion `json:"content"`

}

type SuggestTmpMessage struct {
	Tp string `json:"type"`
	Content []string `json:"content"`

}

type MurderMessageInternal struct {
	CharacterKill string `json:"assassinkill"`
	Rest []PlayerNameMurder `json:"rest"`

}

type SirMessageInternal struct {
	Pick string `json:"pick"`

}

type SirMessage struct {
	Tp string `json:"type"`
	Content SirMessageInternal `json:"content"`

}

type MurderMessage struct {
	Tp string `json:"type"`
	Content MurderMessageInternal `json:"content"`

}

type VoteForJourneyMessage struct {
	Tp string `json:"type"`
	Content VoteForJourney `json:"content"`

}

func getAllBadsChars() []string {
	allBads := make([]string, 0)
	for _, player := range globalBoard.PlayerNames {
		if ch, ok := globalBoard.PlayerToCharacter[player]; ok {
			if _, ok := badCharacters[ch]; ok {
				allBads = append(allBads, ch)
			}
		}
	}
	return allBads
}

func getAllBads() []string {
	allBads := make([]string, 0)
	for _, player := range globalBoard.PlayerNames {
		if ch, ok := globalBoard.PlayerToCharacter[player]; ok {
			if _, ok := badCharacters[ch]; ok {
				allBads = append(allBads, player.Player)
			}
		}
	}
	return allBads
}

func GetMurdersAfterGoodsWins() ([]Murder, bool) {

	murders := make([]Murder, 0)

	if beast, isKingClaudinExists := globalBoard.CharacterToPlayer["The-Questing-Beast"]; isKingClaudinExists  {
		if pellinore, isPrinceClaudinExists := globalBoard.CharacterToPlayer["Pellinore"]; isPrinceClaudinExists  {
			m := Murder{target:[]string{beast.Player}, TargetCharacters:[]string{"The-Questing-Beast"}, By:pellinore.Player}
			murders = append(murders, m)
		}
	}

	if _, isKingClaudinExists := globalBoard.CharacterToPlayer["King-Claudin"]; isKingClaudinExists  {
		if _, isPrinceClaudinExists := globalBoard.CharacterToPlayer["Prince-Claudin"]; isPrinceClaudinExists  {
			if percivalPlayerName, isPercivalExists := globalBoard.CharacterToPlayer["Percival"]; isPercivalExists  {
				m := Murder{target:getAllBads(), TargetCharacters:getAllBadsChars(), By:percivalPlayerName.Player, StateAfterSuccess:VictoryForGood}
				murders = append(murders, m)
			} else if arthurPlayerName, isArthurExists := globalBoard.CharacterToPlayer["King-Arthur"]; isArthurExists  {
				m := Murder{target:getAllBads(), TargetCharacters:getAllBadsChars(), By:arthurPlayerName.Player, StateAfterSuccess:VictoryForGood}
				murders = append(murders, m)
			}
		}
	}

	merlinAppenticePlayerName, ok := globalBoard.CharacterToPlayer["Merlin-Apprentice"]
	initSlice := make([]string, 0)
	targetSlice := make([]string, 0)
	if ok {
		initSlice = append(initSlice, "Merlin-Apprentice")
		targetSlice = append(targetSlice, merlinAppenticePlayerName.Player)
	}
	assassin := globalBoard.CharacterToPlayer["Assassin"]

	if merlinPlayerName, isMerlinExists := globalBoard.CharacterToPlayer["Merlin"]; isMerlinExists  {
		targetSlice = append(targetSlice, merlinPlayerName.Player)
		initSlice = append(initSlice, "Merlin")
	}
	if vivianPlayerName, isVivianExists := globalBoard.CharacterToPlayer["Viviana"]; isVivianExists  {
		targetSlice = append(targetSlice, vivianPlayerName.Player)
		initSlice = append(initSlice, "Viviana")
	}
	if nirlemPlayerName, isNirlemExists := globalBoard.CharacterToPlayer["Nirlem"]; isNirlemExists  {
		targetSlice = append(targetSlice, nirlemPlayerName.Player)
		initSlice = append(initSlice, "Nirlem")
	}

	m := Murder{target:targetSlice, TargetCharacters:initSlice, By:assassin.Player, ByCharacter:"Assassin", StateAfterSuccess:VictoryForBad}
	murders = append(murders, m)

	return murders, len(murders) > 0
}


func GetMurdersAfterBadsWins() ([]Murder, bool) {

	murders := make([]Murder, 0)

	if beast, isKingClaudinExists := globalBoard.CharacterToPlayer["The-Questing-Beast"]; isKingClaudinExists  {
		if pellinore, isPrinceClaudinExists := globalBoard.CharacterToPlayer["Pellinore"]; isPrinceClaudinExists  {
			m := Murder{target:[]string{beast.Player}, TargetCharacters:[]string{"The-Questing-Beast"}, By:pellinore.Player}
			murders = append(murders, m)
		}
	}

	if cordana, isKingClaudinExists := globalBoard.CharacterToPlayer["Cordana"]; isKingClaudinExists  {
		if mordred, isPrinceClaudinExists := globalBoard.CharacterToPlayer["Mordred"]; isPrinceClaudinExists  {
			m := Murder{target:[]string{mordred.Player}, TargetCharacters:[]string{"Cordana"}, By:cordana.Player, StateAfterSuccess:MurdersAfterGoodVictory}
			murders = append(murders, m)
		}
	}

	if kingArthur, isKingArthurExists := globalBoard.CharacterToPlayer["King-Arthur"]; isKingArthurExists  {
		m := Murder{target:getAllBads(), TargetCharacters:getAllBadsChars(), By:kingArthur.Player, StateAfterSuccess:VictoryForGood}
		murders = append(murders, m)
	}

	return murders, len(murders) > 0
}

func GetSecretsFromPlayerName(player PlayerName) []string {

	secrets := make([]string, 0)
	if player.Player == "" {
		return nil
	}

	character := globalBoard.PlayerToCharacter[player]

	if character == "Merlin" {
		for k, v := range globalBoard.CharacterToPlayer {

			if _, ok := badCharacters[k]; ok && k != "Mordred" && k != "Accolon" {
				if k == "Oberon" {
					secrets = append(secrets, v.Player + " is Oberon")
				} else {
					secrets = append(secrets, v.Player + " is bad")
				}
			}
			if k == "Lot" {
				secrets = append(secrets, v.Player + " is Lot")
			}
			if k == "Ginerva" {
				secrets = append(secrets, v.Player + " is bad")
			}
			if k == "Sir-Kay" {
				secrets = append(secrets, v.Player + " is bad")
			}
			if k == "Gawain" {
				secrets = append(secrets, v.Player + " is Gawain")
			}
		}
	}
	if _, ok := goodCharacters[character]; ok && character != "Nirlem" && character != "Lot" {
		if player, ok := globalBoard.CharacterToPlayer["Nirlem"]; ok && character != "Lancelot-Good" {
			secrets = append(secrets, player.Player + " is Nirlem")
		}
	}
	if character == "Guinevere" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Lancelot-Good" {
				secrets = append(secrets, v.Player + " is Lancelot")
			}
			if k == "Lancelot-Bad" {
				secrets = append(secrets, v.Player + " is Lancelot")
			}
		}
	}
	if character == "Iseult" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Tristan" {
				secrets = append(secrets, v.Player + " is Tristan")
			}
		}
	}
	if character == "Prince-Claudin" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "King-Claudin" {
				secrets = append(secrets, v.Player + " is King-Claudin")
			}
		}
	}
	if character == "King-Claudin" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Prince-Claudin" {
				secrets = append(secrets, v.Player + " is Prince-Claudin")
			}
		}
	}

	if character == "Merlin-Apprentice" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Percival" {
				secrets = append(secrets, v.Player + " is Percival/Assasin")
			}
			if k == "Assassin" {
				secrets = append(secrets, v.Player + " is Percival/Assassin")
			}
		}
	}
	if character == "Tristan" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Iseult" {
				secrets = append(secrets, v.Player + " is Iseult")
			}
		}
	}
	if character == "Lot" {
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; ok && k != character && k != "Oberon" && k != "Accolon" {
				if k == "Polygraph" {
					secrets = append(secrets, v.Player+" is polygraph")
				} else {
					secrets = append(secrets, v.Player+" is bad")
				}
			}
		}
	}
	if character == "Nimue" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Galahad" {
				secrets = append(secrets, v.Player + " is Galahad")
			}
			if k == "Merlin" {
				secrets = append(secrets, v.Player + " is Merlin")
			}
		}
	}
	if character == "Morgana" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Gawain" {
				secrets = append(secrets, v.Player + " is Gawain")
			}
		}
	}
	if character == "Percival" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Morgana" {
				if _, ok := globalBoard.CharacterToPlayer["Merlin"]; !ok {
					secrets = append(secrets, v.Player + " is Morgana/Viviana")
				} else {
					secrets = append(secrets, v.Player+" is Morgana/Merlin")
				}
			}
			if k == "Merlin" {
				secrets = append(secrets, v.Player + " is Morgana/Merlin")
			}
			if k == "Viviana" {
				if _, ok := globalBoard.CharacterToPlayer["Merlin"]; !ok {
					secrets = append(secrets, v.Player + " is Morgana/Viviana")
				}
			}

		}
	}
	if _, ok := badCharacters[character] ; ok && character != "Oberon" && character != "Accolon" && character != "Lancelot-Bad" {
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; ok && k != character && k != "Oberon" && k != "Accolon" {
				if k == "Polygraph" {
					secrets = append(secrets, v.Player+" is polygraph")
				} else {
					secrets = append(secrets, v.Player+" is bad")
				}
			}
		}
	}

	if character == "The-Questing-Beast" {
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Pellinore" {
				secrets = append(secrets, v.Player + " is Pellinore")
			}
		}
	}

	if character == "Gawain" {
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; ok && k != character && k != "Oberon" && k != "Accolon" {
				secrets = append(secrets, v.Player+" ")
			}
			if k == "Percival" || k == "Merlin" || k == "Nirlem" || k == "Viviana" {
				secrets = append(secrets, v.Player+" ")
			}
		}
	}

	return secrets
}

func getVoteStr(vote int) string {
	if 0 == vote { return "Fail" }
	if 1 == vote { return "Success" }
	if 2 == vote { return "Reversal" }
	if 3 == vote { return "Beast" }
	return "N/A"
}
func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}
func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num * output)) / output
}

func ExcaliburHandler(excaliburPick []string) {
	log.Println("got new excalibur pick:", excaliburPick)
	globalMutex.Lock()
	current := globalBoard.quests.current
	mp := globalBoard.quests.playersVotes[current]
	res := globalBoard.quests.results[current+1]
	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	if len(excaliburPick) == 1 {

		character := globalBoard.PlayerToCharacter[PlayerName{excaliburPick[0]}]
		playerVote := globalBoard.quests.playerVotedForCurrent[excaliburPick[0]]
		curEntry.ExcaliburChosenPlayer = excaliburPick[0]
		var newVote int
		log.Println("character:", character, "player vote:", playerVote)
		globalBoard.suggestions.excalibur.ChosenPlayerVote = playerVote
		globalBoard.Secrets[globalBoard.suggestions.excalibur.Player] = append(globalBoard.Secrets[globalBoard.suggestions.excalibur.Player], excaliburPick[0]+" voted " + getVoteStr(playerVote)+"(Quest "+strconv.FormatFloat(float64(curEntry.Id), 'f', 2, 32)+")")
		if playerVote == 2 {
			res.NumOfReversal--
			curEntry.NumberOfReversal--
			if character == "Good-Angel" {
				res.NumOfFailures++
				curEntry.NumberOfFailures++
				log.Println("new vote fail")
				newVote = 0 /*Fail*/
			} else if character == "Bad-Angel" {
				res.NumOfSuccess++
				curEntry.NumberOfSuccesses++
				log.Println("new vote success")
				newVote = 1 /*Success*/
			}
		} else if playerVote == 0 || playerVote == 3 {
			if playerVote == 0 {
				res.NumOfFailures--
				curEntry.NumberOfFailures--
			} else {
				res.NumOfBeasts--
				curEntry.NumberOfBeasts--
			}
			newVote = 1
			log.Println("new vote success")
			res.NumOfSuccess++
			curEntry.NumberOfSuccesses++
		} else if playerVote == 1 {
			res.NumOfSuccess--
			curEntry.NumberOfSuccesses--
			curEntry.NumberOfFailures++
			res.NumOfFailures++
			newVote = 0
			log.Println("new vote fail")
		}
		for i, vote := range mp {
			if vote == playerVote {
				mp[i] = newVote
				break
			}
		}
		globalBoard.quests.playerVotedForCurrent[excaliburPick[0]] = newVote
	}

	EndJourney(&res, mp, &curEntry, current)
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry
	globalBoard.quests.results[current+1] = res
	globalBoard.quests.playersVotes[current] = mp
	globalBoard.quests.current++
	globalMutex.Unlock()
}


func StartGameHandler(newGameConfig GameConfiguration) {
	log.Println("newGameConfig", newGameConfig)
	globalMutex.Lock()

	chosenCharacters := make([]string, 0)
	numOfPlayers := len(globalBoard.PlayerNames)
	requiredBads := globalConfigPerNumOfPlayers[numOfPlayers].NumOfBadCharacters

	if newGameConfig.Excalibur == true {
		globalBoard.quests.Flags[EXCALIBUR] = true
		log.Println("excalibur - on ")
	}
	globalBoard.lancelotCards = []int{0,0,1,0,1,0,0}
	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(globalBoard.lancelotCards), func(i, j int) {
		globalBoard.lancelotCards[i], globalBoard.lancelotCards[j] = globalBoard.lancelotCards[j], globalBoard.lancelotCards[i]
	})
	log.Println("===========", globalBoard.lancelotCards)
	var numOfBads int
	var numOfGood int
	for _, v := range newGameConfig.Characters {
		if v.Checked == true {
			if badCharacters[v.Name] == true {
				numOfBads++
			} else if goodCharacters[v.Name] == true {
				numOfGood++
			} else if v.Name == "Puck" {
				numOfGood++
			} else if v.Name == "Ginerva" || v.Name == "Gawain" {
				numOfBads++
			} else {
				globalMutex.Unlock()
				return
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

	chosenCharacters, assassinPlayer := assignCharactersToRegisteredPlayers(newGameConfig.Characters, chosenCharacters)
	if chosenCharacters == nil {
		log.Fatal("No assassin chosen")
	}

	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(chosenCharacters), func(i, j int) {
		chosenCharacters[i], chosenCharacters[j] = chosenCharacters[j], chosenCharacters[i]
	})
	log.Println("ttttttttttttttttt", chosenCharacters)
	if _, ok := globalBoard.CharacterToPlayer["Seer"]; ok {
		globalBoard.State = SirPickPlayer
	} else {
		globalBoard.State = WaitingForSuggestion
	}
	if globalBoard.quests.results == nil {
		globalBoard.quests.results = make(map[int]QuestStats)
	}

	globalBoard.Characters = chosenCharacters
	for i:=0; i < globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests; i++ {
		en := QuestStats{}
		log.Println(i)
		en.Ppp = getTypeOfLevel(i+1, len(globalBoard.PlayerNames))
		en.NumOfPlayers = globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].PlayersPerLevel[i]
		globalBoard.quests.results[i+1] = en
		log.Println(en)
	}
	globalBoard.suggestions.suggesterIndex = 0

	numOfUnsuccesfulRetries:= globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].RetriesPerLevel[globalBoard.quests.current]
	suggesterVetoIn := (globalBoard.suggestions.suggesterIndex+numOfUnsuccesfulRetries-1) % len(globalBoard.PlayerNames)
	globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[suggesterVetoIn].Player

	for _, player := range globalBoard.PlayerNames {
		globalBoard.Secrets[player.Player] = GetSecretsFromPlayerName(player)
		log.Println(player, "      =     ", globalBoard.Secrets[player.Player])
	}

	_, hasBadLancelot := globalBoard.CharacterToPlayer["Lancelot-Bad"]
	_, hasGoodLancelot := globalBoard.CharacterToPlayer["Lancelot-Good"]
	if hasBadLancelot && hasGoodLancelot {
		globalBoard.quests.Flags[HAS_TWO_LANCELOT] = true
	} else if hasBadLancelot {
		globalBoard.quests.Flags[HAS_ONLY_BAD_LANCELOT] = true
	} else if hasGoodLancelot {
		globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] = true
	}

	globalBoard.CharacterToPlayer["Assassin"] = PlayerName{assassinPlayer}
	globalMutex.Unlock()
}

type SecretResponse struct {
	Character string `json:"character,omitempty"`
	Secrets []string `json:"secrets,omitempty"`
	PlayersWithGoodCharacter []string `json:"goodplayers,omitempty"`//for vivian
	PlayersWithBadCharacter []string `json:"badplayers,omitempty"` //for vivian
}

func GetNightSecretsFromPlayerName(player PlayerName) SecretResponse {
	response := SecretResponse{}

	if player.Player == "" {
		return SecretResponse{}
	}

	character := globalBoard.PlayerToCharacter[player]
	response.Character = character
	if character == "Viviana" {
		log.Println("#########################################")
		log.Println(globalBoard.playersWithBadCharacter)
		log.Println(globalBoard.playersWithGoodCharacter)
		response.PlayersWithBadCharacter = globalBoard.playersWithBadCharacter
		response.PlayersWithGoodCharacter = globalBoard.playersWithGoodCharacter
		log.Println(response.PlayersWithBadCharacter )
		log.Println(response.PlayersWithGoodCharacter)
		response.Secrets = globalBoard.Secrets[player.Player]
	} else if secrets, ok := globalBoard.Secrets[player.Player]; ok {
		return SecretResponse{Character: character, Secrets: secrets}
	}
	return response
}

func assignCharactersToRegisteredPlayers(newGameConfig []Ch, chosenCharacters []string) ([]string, string) {
	var assassinCharacter string
	for _, v := range newGameConfig {
		if v.Checked == true {
			chosenCharacters = append(chosenCharacters, v.Name)
			if v.Assassin == true {
				assassinCharacter = v.Name
			}
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
	if _, ok := globalBoard.CharacterToPlayer["Assassin"]; ok {
		assassinCharacter = "Assassin"
	}
	return chosenCharacters, globalBoard.CharacterToPlayer[assassinCharacter].Player
}

type ListOfPlayersResponse struct {
	Total int `json:"total,omitempty"`
	Players []PlayerName `json:"all,omitempty"`
}

func getTypeOfLevel(levelNum int, numOfPlayers int) int {
	log.Println(levelNum, numOfPlayers)
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

type Suggestion struct {
	Players []string `json:"players,omitempty"`
	ExcaliburPlayer string `json:"excalibur,omitempty"`
}

type ListOfSuggestions2 struct {
	Players []string `json:"all,omitempty"`
}

type PlayerName2 struct {
	Player string `json:"player,omitempty"`
	Ch bool `json:"ch,omitempty"`
}

type PlayerNameMurder struct {
	Player string `json:"player,omitempty"`
	Ch bool `json:"ch,omitempty"`
	CharacterToKill string `json:"characterToKill,omitempty"`
}

func sameStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y] -= 1
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

func HandleSir(m SirMessageInternal) {
	globalMutex.Lock()
	pick := m.Pick
	character := globalBoard.PlayerToCharacter[PlayerName{pick}]
	SirPlayer := globalBoard.CharacterToPlayer["Seer"]
	globalBoard.Secrets[SirPlayer.Player] = append(globalBoard.Secrets[SirPlayer.Player], pick + " is " + character)
	globalBoard.State = WaitingForSuggestion
	globalMutex.Unlock()
}

func HandleMurder(m MurderMessageInternal) {
	var curMurder Murder
	selection := m.Rest
	characterToKill := m.CharacterKill
	log.Println("selection: ", selection)
	if globalBoard.State == MurdersAfterGoodVictory {
		curMurder = globalBoard.PendingMurders[0]
	} else if globalBoard.State == MurdersAfterBadVictory {
		curMurder = globalBoard.PendingMurders[0]
	}

	chosenPlayers := make([]string, 0)
	for _, player := range selection {
		if player.Ch {
			chosenPlayers = append(chosenPlayers, player.Player)
			murderInfo, ok := globalBoard.PlayerToMurderInfo[player.Player]
			if ok {
				murderInfo.by = append(murderInfo.by, curMurder.By)
				globalBoard.PlayerToMurderInfo[player.Player] = murderInfo
			} else {
				globalBoard.PlayerToMurderInfo[player.Player] = MurderInfo{by:[]string{curMurder.By}}
			}
		}
	}

	globalBoard.PendingMurders = globalBoard.PendingMurders[1:]
	murderResult := MurderResult{targetCharacter:curMurder.TargetCharacters, byCharacter:curMurder.ByCharacter}

	var isSuccess bool
	if curMurder.ByCharacter == "Assassin" && len(chosenPlayers)==1 {
		for _, v := range curMurder.target {
			if v == chosenPlayers[0] {
				isSuccess = true
				log.Println("assassin murder success. chosenPlayers ", chosenPlayers[0])
			}
		}
		if (globalBoard.PlayerToCharacter[PlayerName{chosenPlayers[0]}] != characterToKill) {
			log.Println("assassin murder failed. chosen Player is ",chosenPlayers[0], " with role ", globalBoard.PlayerToCharacter[PlayerName{chosenPlayers[0]}], "instead of ", characterToKill)
		}
	} else {
		isSuccess = sameStringSlice(curMurder.target, chosenPlayers)
	}

	if isSuccess {
		//murder succeeded!
		log.Println("Murder Success! Killer:", curMurder.By, " Selection: ", selection)
		murderResult.success = true
		murderResult.byCharacter = curMurder.ByCharacter
		murderResult.target = chosenPlayers
		if curMurder.StateAfterSuccess != 0 {
			oldState := globalBoard.State
			globalBoard.State = curMurder.StateAfterSuccess
			log.Println("New State:", globalBoard.State)
			if oldState == MurdersAfterBadVictory && globalBoard.State == MurdersAfterGoodVictory {
				pendingMurders, hasMurders := GetMurdersAfterGoodsWins()
				if !hasMurders {
					globalBoard.State = VictoryForGood
					globalBoard.PendingMurders = make([]Murder, 0)
					return
				} else {
					globalBoard.PendingMurders = pendingMurders
				}
			}
		}
	} else {
		murderResult.success = false
	}

	if len(globalBoard.PendingMurders) == 0 {
		log.Println("No more murders")
		if globalBoard.State == MurdersAfterGoodVictory {
			globalBoard.State = VictoryForGood
		} else if globalBoard.State == MurdersAfterBadVictory {
			globalBoard.State = VictoryForBad
		}
	}
}


func HandleNewSuggest(pl Suggestion) {
	globalMutex.Lock()
	suggestedPlayers := pl.Players
	suggestedCharacters := make(map[string]bool, 0)

	for _, v := range pl.Players {
			suggestedCharacters[globalBoard.PlayerToCharacter[PlayerName{v}]] = true
	}

	suggesterIn := globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	newEntry := QuestArchiveItem{Id:globalBoard.QuestStage, Suggester:globalBoard.PlayerNames[suggesterIn], SuggestedPlayers:suggestedPlayers, ExcaliburPlayer:pl.ExcaliburPlayer}

	log.Println("SuggestedPlayers:", suggestedPlayers, ",ExcaliburPlayer:", pl.ExcaliburPlayer, ",Suggester:", globalBoard.PlayerNames[suggesterIn].Player)
	globalBoard.suggestions.SuggestedTemporaryPlayers = ""
	globalBoard.suggestions.SuggestedPlayers = suggestedPlayers
	globalBoard.suggestions.SuggestedCharacters = suggestedCharacters
	globalBoard.suggestions.excalibur.Player = pl.ExcaliburPlayer
	globalBoard.suggestions.excalibur.Suggester = globalBoard.PlayerNames[suggesterIn].Player

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

		if "Sir-Kay" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
		} else if "Mordred" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
		} else if "Lot" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			vivianaSecrets := make([]string, 1)
			vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player + " is Lot")
			globalBoard.Secrets["Viviana"] = vivianaSecrets
		} else if "Gawain" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			vivianaSecrets := globalBoard.Secrets["Viviana"]
			if vivianaSecrets == nil {
				vivianaSecrets = make([]string, 1)
			}
			vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player + " is Gawain")
			globalBoard.Secrets["Viviana"] = vivianaSecrets
		} else if "Ginerva" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			vivianaSecrets := globalBoard.Secrets["Viviana"]
			if vivianaSecrets == nil {
				vivianaSecrets = make([]string, 1)
			}
			vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player + " is Gawain")
			globalBoard.Secrets["Viviana"] = vivianaSecrets
		} else if _, isSuggesterBadCharacter := badCharacters[globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]]]; isSuggesterBadCharacter {
			log.Println("suggester is bad")
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
		} else {
			log.Println("suggester is good")
			globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
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


func HandleTemporarySuggest(pl []string) {
	globalMutex.Lock()
	suggestedPlayersStr := ""

	for i, v := range pl {
		if i > 0 {
			suggestedPlayersStr += " ,"
		}
		suggestedPlayersStr += v
	}

	globalBoard.suggestions.SuggestedTemporaryPlayers = suggestedPlayersStr
	globalMutex.Unlock()

}



type PlayerInfo struct {
	Character string `json:"ch,omitempty"`
	IsKilled bool `json:"isKilled,omitempty"`
	isKilledBy []string
}

type SirPick struct {
	Options []string  `json:"options,omitempty"`
	Pick string  `json:"pick,omitempty"`
}

type Excalibur struct {
	Player string  `json:"excalibur_player,omitempty"`
	Suggester string  `json:"suggester,omitempty"`
	ChosenPlayerVote int  `json:"vote,omitempty"`
}

type GameState struct {
	Players ListOfPlayersResponse `json:"players,omitempty"`
	CurrentQuest int `json:"current,omitempty"`
	Characters map[string]CharacterDescription `json:"characters,omitempty"`
	Size int `json:"size,omitempty"`
	State int `json:"state"`
	Archive []QuestArchiveItem `json:"archive"`
	Secrets SecretResponse `json:"secrets"`
	Suggester string `json:"suggester,omitempty"`
	Murder Murder `json:"murder,omitempty"`
	SirPick SirPick `json:"sir,omitempty"`
	OptionalVotes []string `json:"optionalVotes,omitempty"`
	SuggesterVeto string `json:"suggesterVeto,omitempty"`
	SuggestedPlayers []string `json:"suggestedPlayers,omitempty"`
	SuggestedTemporaryPlayers string `json:"suggestedTemporaryPlayers,omitempty"`
	PlayersVotedForCurrQuest []string `json:"PlayersVotedForCurrQuest,omitempty"`
	PlayersVotedYes []string `json:"PlayersVotedYesForSuggestion,omitempty"`
	PlayersVotedNo []string `json:"PlayersVotedNoForSuggestion,omitempty"`
	Results map[int]QuestStats `json:"results,omitempty"`
	PlayerInfo map[string]PlayerInfo `json:"playerToCharacters,omitempty"`
	IsExcalibur bool `json:"excalibur,omitempty"`
	SuggestedExcalibur string `json:"suggestedExcalibur,omitempty"`
}

func GetGameState(clientId string) GameState  {
	globalMutex.RLock()
	board := GameState{}

	if globalBoard.State == SirPickPlayer && globalBoard.CharacterToPlayer["Seer"].Player == clientId {
		var index int
		for i, p := range globalBoard.PlayerNames {
			if p.Player == clientId {
				index = i
			}
		}
		options := []string{globalBoard.PlayerNames[(index-1) % len(globalBoard.PlayerNames)].Player, globalBoard.PlayerNames[(index+1) % len(globalBoard.PlayerNames)].Player}
		board.SirPick = SirPick{Options:options}
	}
	if globalBoard.quests.Flags[EXCALIBUR] {
		board.IsExcalibur = true
		board.SuggestedExcalibur = globalBoard.suggestions.excalibur.Player
	}
	board.SuggestedTemporaryPlayers = globalBoard.suggestions.SuggestedTemporaryPlayers
	board.Players.Total = len(globalBoard.PlayerNames)
	board.Players.Players = globalBoard.PlayerNames
	board.SuggestedPlayers = globalBoard.suggestions.SuggestedPlayers
	board.CurrentQuest = globalBoard.quests.current + 1
	board.CurrentQuest = globalBoard.quests.current + 1
	board.Size = globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests
	board.Results = globalBoard.quests.results
	board.Characters = make(map[string]CharacterDescription)
	for _, ch := range globalBoard.Characters {
		board.Characters[ch]=CharactersDescriptionMap[ch]
	}
	board.PlayersVotedForCurrQuest = globalBoard.quests.playerVotedForCurrentQuest
	board.SuggesterVeto = globalBoard.suggestions.PlayerWithVeto
	cpy := make([]QuestArchiveItem, len(globalBoard.archive))
	copy(cpy, globalBoard.archive)
	if len(cpy) > 0 && globalBoard.State == SuggestionVoting {
		cpy[len(cpy)-1].PlayersVotedYes = make([]string, 0)
		cpy[len(cpy)-1].PlayersVotedNo = make([]string, 0)
	}
	if len(cpy) > 0 && (globalBoard.State == JorneyVoting || globalBoard.State == ExcaliburPick) {
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
	if globalBoard.State == MurdersAfterBadVictory || globalBoard.State == MurdersAfterGoodVictory {
		board.Murder.TargetCharacters = globalBoard.PendingMurders[0].TargetCharacters
		board.Murder.By = globalBoard.PendingMurders[0].By
		board.Murder.ByCharacter = globalBoard.PendingMurders[0].ByCharacter
	}
	if globalBoard.State == VictoryForGood || globalBoard.State == VictoryForBad {
		board.PlayerInfo = make(map[string]PlayerInfo)
		for _, pl := range board.Players.Players {
			playerInfo := PlayerInfo{}
			playerInfo.Character = globalBoard.PlayerToCharacter[pl]
			_, isKilled := globalBoard.PlayerToMurderInfo[pl.Player]
			playerInfo.IsKilled = isKilled
			board.PlayerInfo[pl.Player] = playerInfo
		}
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

	if globalBoard.PlayerToCharacter[PlayerName{vote.PlayerName}] == "The-Questing-Beast" && vote.Vote==1 {
		globalBoard.quests.Flags[BEAST_FIRST_SUCCESS] = true
	}

	origVote := vote.Vote
	if origVote == 3 {
		origVote = 0
	}
	log.Println(vote.PlayerName, " voted ", vote.Vote)
	globalBoard.quests.playerVotedForCurrentQuest = append(globalBoard.quests.playerVotedForCurrentQuest, vote.PlayerName)
	globalBoard.quests.playerVotedForCurrent[vote.PlayerName] = origVote
	mp := append(globalBoard.quests.playersVotes[current], origVote)
	requiredVotes := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].PlayersPerLevel[current]

	res := globalBoard.quests.results[current+1]

	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	if vote.Vote == 0 {
		res.NumOfFailures++
		curEntry.NumberOfFailures++
	} else if vote.Vote == 1 {
		res.NumOfSuccess++
		curEntry.NumberOfSuccesses++
	} else if vote.Vote == 3 {
		res.NumOfBeasts++
		curEntry.NumberOfBeasts++
	} else {
		res.NumOfReversal++
		curEntry.NumberOfReversal++
	}

	if len(mp) == requiredVotes { //last vote
		if _, ok := globalBoard.quests.Flags[EXCALIBUR]; ok  {
			globalBoard.State = ExcaliburPick
			//update info
			globalBoard.archive[len(globalBoard.archive)-1] = curEntry
			globalBoard.quests.results[current+1] = res
			globalBoard.quests.playersVotes[current] = mp
			globalMutex.Unlock()
			return
		}
		EndJourney(&res, mp, &curEntry, current)
	}
	//update info
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry
	globalBoard.quests.results[current+1] = res
	globalBoard.quests.playersVotes[current] = mp
	if _, ok := globalBoard.quests.Flags[EXCALIBUR]; !ok && len(mp) == requiredVotes { //last vote
		globalBoard.quests.current++
	}
	globalMutex.Unlock()
}

func EndJourney(res *QuestStats, mp []int, curEntry *QuestArchiveItem, current int) {
	retriesPerLevel := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].RetriesPerLevel
	if globalBoard.quests.current+1 < len(retriesPerLevel) { //not last quest in game
		numOfUnsuccesfulRetries := retriesPerLevel[globalBoard.quests.current+1]
		suggesterVetoIn := (globalBoard.suggestions.suggesterIndex + numOfUnsuccesfulRetries - 1) % len(globalBoard.PlayerNames)
		globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[suggesterVetoIn].Player
	}
	res.Final = CalculateQuestResult(mp)
	log.Println("Quest Result:(", globalBoard.quests.current+1, ")", res.Final)
	curEntry.FinalResult = res.Final
	globalBoard.quests.results[current+1] = *res
	if globalBoard.quests.results[current+1].Final == JorneySuccess {
		globalBoard.quests.successfulQuest++
	} else {
		globalBoard.quests.unsuccessfulQuest++
	}
	if playerName, ok := globalBoard.CharacterToPlayer["King-Arthur"]; ok {
		//King-Arthur is playing
		if vote, ok := globalBoard.quests.playerVotedForCurrent[playerName.Player]; ok {
			//King-Arthur was in this quest
			log.Println("switch King-Arthur's \"Fail\" to \"Success")
			realResults := make([]int, len(mp))
			copy(realResults, mp)
			for i, res := range realResults {
				if res == vote {
					realResults[i] = (1+ vote) % 2
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
	numOfExpectedQuests := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests
	if globalBoard.quests.successfulQuest > numOfExpectedQuests/2 {
		pendingMurders, hasMurders := GetMurdersAfterGoodsWins()
		if !hasMurders {
			globalBoard.State = VictoryForGood
		} else {
			fmt.Println(pendingMurders)
			globalBoard.State = MurdersAfterGoodVictory
			globalBoard.PendingMurders = pendingMurders
		}
	} else if globalBoard.quests.unsuccessfulQuest > numOfExpectedQuests/2 || (numOfExpectedQuests == 4 && globalBoard.quests.unsuccessfulQuest == numOfExpectedQuests/2) {
		pendingMurders, hasMurders := GetMurdersAfterBadsWins()
		if !hasMurders {
			globalBoard.State = VictoryForBad
		} else {
			globalBoard.State = MurdersAfterBadVictory
			globalBoard.PendingMurders = pendingMurders
		}
	} else { //game continue
		if globalBoard.quests.Flags[HAS_TWO_LANCELOT] ||
			globalBoard.quests.Flags[HAS_ONLY_BAD_LANCELOT] ||
			globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] {
			//random number to decide if lancelots switch
			isSwitchLancelots := globalBoard.lancelotCards[globalBoard.lancelotCardsIndex]
			globalBoard.lancelotCardsIndex = (globalBoard.lancelotCardsIndex + 1) % len(globalBoard.lancelotCards)
			if isSwitchLancelots == 1 {
				if globalBoard.quests.Flags[HAS_TWO_LANCELOT] {
					lanBad := globalBoard.CharacterToPlayer["Lancelot-Bad"]
					lanGood := globalBoard.CharacterToPlayer["Lancelot-Good"]
					globalBoard.CharacterToPlayer["Lancelot-Bad"] = lanGood
					globalBoard.CharacterToPlayer["Lancelot-Good"] = lanBad
					globalBoard.PlayerToCharacter[lanBad] = "Lancelot-Good"
					globalBoard.PlayerToCharacter[lanGood] = "Lancelot-Bad"
					curEntry.IsSwitchLancelot = true
					//fix bug of viviana that seeother lanselot
					/*for i, pl := range globalBoard.playersWithBadCharacter {
						if pl == lanBad.Player {
							globalBoard.playersWithBadCharacter[i] = lanGood.Player
							break
						}
					}
					for i, pl := range globalBoard.playersWithGoodCharacter {
						if pl == lanGood.Player {
							globalBoard.playersWithGoodCharacter[i] = lanBad.Player
							break
						}
					}*/
				} else if globalBoard.quests.Flags[HAS_ONLY_BAD_LANCELOT] {
					lanBad := globalBoard.CharacterToPlayer["Lancelot-Bad"]
					globalBoard.CharacterToPlayer["Lancelot-Good"] = lanBad
					globalBoard.PlayerToCharacter[lanBad] = "Lancelot-Good"
					delete(globalBoard.CharacterToPlayer, "Lancelot-Bad")
					for i, ch := range globalBoard.Characters {
						if ch == "Lancelot-Bad" {
							globalBoard.Characters = append(globalBoard.Characters[:i], globalBoard.Characters[i+1:]...)
						}
						globalBoard.Characters = append(globalBoard.Characters, "Lancelot-Good")
						break
					}
					curEntry.IsSwitchLancelot = true
					delete(globalBoard.quests.Flags, HAS_ONLY_BAD_LANCELOT)
					globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] = true
				} else if globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] {
					lanBad := globalBoard.CharacterToPlayer["Lancelot-Good"]
					globalBoard.CharacterToPlayer["Lancelot-Bad"] = lanBad
					globalBoard.PlayerToCharacter[lanBad] = "Lancelot-Bad"
					delete(globalBoard.CharacterToPlayer, "Lancelot-Good")
					for i, ch := range globalBoard.Characters {
						if ch == "Lancelot-Good" {
							globalBoard.Characters = append(globalBoard.Characters[:i], globalBoard.Characters[i+1:]...)
						}
						globalBoard.Characters = append(globalBoard.Characters, "Lancelot-Bad")
						break
					}
					curEntry.IsSwitchLancelot = true
					delete(globalBoard.quests.Flags, HAS_ONLY_GOOD_LANCELOT)
					globalBoard.quests.Flags[HAS_ONLY_BAD_LANCELOT] = true
				}
			}
		}
		//end of special actions after quest
		globalBoard.State = WaitingForSuggestion
	}
	globalBoard.quests.playerVotedForCurrentQuest = make([]string, 0)
	globalBoard.quests.playerVotedForCurrent = make(map[string]int)
	globalBoard.votesForNextMission = make(map[string]bool) //for suggestions
	globalBoard.suggestions.SuggestedPlayers = make([]string, 0)
}

func CalculateQuestResult(mp []int) int {
	result := JorneySuccess
	log.Println("++ last")
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
		log.Println("vote is over. num of players =", len(globalBoard.PlayerNames))
		curEntry.IsSuggestionOver = true

		numOfQuests := globalConfigPerNumOfPlayers[len(globalBoard.PlayerNames)].NumOfQuests
		if globalBoard.quests.current+1 == numOfQuests { //last quest in game
			if gawainPlayer, ok := globalBoard.CharacterToPlayer["Gawain"]; ok {
				for _, c := range globalBoard.suggestions.SuggestedPlayers {
					if c == gawainPlayer.Player {
						if gaVote, ok := globalBoard.votesForNextMission[gawainPlayer.Player]; ok {
							if gaVote {
								globalBoard.isSuggestionGood++
							} else {
								globalBoard.isSuggestionBad++
							}
						}
					}
				}

			}
		}

		if globalBoard.isSuggestionGood > globalBoard.isSuggestionBad {

			if gawainPlayer, ok := globalBoard.CharacterToPlayer["Gawain"]; ok {
				if globalBoard.quests.current+1 == numOfQuests {
					for _, c := range globalBoard.suggestions.SuggestedPlayers {
						if c == gawainPlayer.Player {
							globalBoard.State = VictoryForGawain
							globalMutex.Unlock()
							return
						}
					}
				}
			}
			globalBoard.State = JorneyVoting
			globalBoard.suggestions.unsuccessfulRetries = 0
			curEntry.IsSuggestionAccepted = true
			globalBoard.QuestStage += 0.01
			globalBoard.QuestStage = float32(math.Ceil(float64(globalBoard.QuestStage)))

			//for vivian
			if "Sir-Kay" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			} else if "Mordred" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			} else if "Lot" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
				vivianaSecrets := globalBoard.Secrets["Viviana"]
				if vivianaSecrets == nil {
					vivianaSecrets = make([]string, 1)
				}
				vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player + " is Lot")
				globalBoard.Secrets["Viviana"] = vivianaSecrets
			} else if "Gawain" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
				vivianaSecrets := globalBoard.Secrets["Viviana"]
				if vivianaSecrets == nil {
					vivianaSecrets = make([]string, 1)
				}
				vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player + " is Gawain")
				globalBoard.Secrets["Viviana"] = vivianaSecrets
			} else if "Ginerva" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
				vivianaSecrets := globalBoard.Secrets["Viviana"]
				if vivianaSecrets == nil {
					vivianaSecrets = make([]string, 1)
				}
				vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player + " is Gawain")
				globalBoard.Secrets["Viviana"] = vivianaSecrets
			} else if _, isSuggesterBadCharacter := badCharacters[globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)]]]; isSuggesterBadCharacter {
				log.Println("suggester is bad")
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			} else {
				log.Println("suggester is good")
				globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)].Player)
			}

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
	log.Println("err:", err)
	if err != nil {
		log.Println(err)
		http.Error(res, "Request failed!", http.StatusUnauthorized)
		return
	}
	log.Println("err")
	data := claims.Claims.(*JWTData)

	userName := data.CustomClaims["userName"]

	conn, error := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(res, req, nil)
	if error != nil {
		log.Println("err2")
		http.NotFound(res, req)
		return
	}
	log.Println("wow2")


	//uuid,_:= uuid.NewV4()
	client := &Client{id: userName, socket: conn, send: make(chan []byte)}
	log.Println("wow3")
	globalBoard.manager.register <- client

	log.Println("start threads")
	go client.read()
	go client.write()

}

type userRouter struct {
	userService *UserService1
}

func main() {
	fmt.Println("Starting server at http://localhost:12345...")
	f, _ := os.OpenFile("testlogfile.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	log.SetOutput(f)

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

	router.HandleFunc("/register2", userRouter.createUserHandler).Methods("PUT", "OPTIONS", "POST")
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
		log.Println("error", err)
		return
	}
	log.Println(user)
	err = ur.userService.Create(&user)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("good")
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

		log.Println(err)
		return
	}
	c := Hash{}
	log.Println("login start")
	compareError := c.Compare(dbUser.Password, user.Password)
	if compareError == nil {
		claims := JWTData{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour*300).Unix(),
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
	log.Println("login end")
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