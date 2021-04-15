package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
)


type ListOfPlayersResponse struct {
	Total   int          `json:"total,omitempty"`
	Players []PlayerName `json:"all,omitempty"`
	Active []PlayerName `json:"active,omitempty"`
}

type Message struct {
	Type      string `json:"ty"`
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}

type PlayerGone struct {
	Type    string                `json:"ty"`
	Players ListOfPlayersResponse `json:"players"`
}

func (manager *ClientManager) start() {
	defer func() {
		//debug.PrintStack()
		log.Println("manager start thread termination. fatal error")
		//buf := make([]byte, 1<<16)
	//	runtime.Stack(buf, true)
		//log.Println("%s", buf)
	}()
	for {
		log.Println("inside MANAGER loop")
		select {
		case conn := <-manager.register:
			log.Println("register new connection")
			if _, ok := manager.clients[conn]; !ok {
				globalMutex.Lock()
				found := false
				for _, v := range globalBoard.PlayerNames {
					if v.Player == conn.id {
						found = true
					}
				}
				if !found && (globalBoard.State == NotStarted || (globalBoard.State >= VictoryForGood && globalBoard.State <= VictoryForGawain)) {
					log.Println("Adding", conn.id, " to player names list")
					globalBoard.PlayerNames = append(globalBoard.PlayerNames, PlayerName{conn.id})
				}

				globalMutex.Unlock()
				manager.clients[conn] = true
				jsonMessage, _ := json.Marshal(&Message{Content: "/A new socket has connected."})
				manager.send(jsonMessage, conn)
			}
		case conn := <-manager.unregister:
			log.Println("unregister connection")
			if _, ok := manager.clients[conn]; ok {
				globalMutex.Lock()
				//playerName, ok := globalBoard.clientIdToPlayerName[conn.id]
				if ok {
					log.Println("unregister ", conn.id)
					if globalBoard.State == NotStarted || (globalBoard.State >= VictoryForGood && globalBoard.State <= VictoryForGawain) {
						index := SliceIndex(len(globalBoard.PlayerNames), func(i int) bool { return globalBoard.PlayerNames[i] == PlayerName{conn.id} })
						if index >= 0 {
							globalBoard.PlayerNames = removePlayer(globalBoard.PlayerNames, index)
							log.Println(conn.id, " was removed for player names list: ", globalBoard.PlayerNames)
						}
					}

					delete(globalBoard.clientIdToPlayerName, conn.id)
				}

				ls := ListOfPlayersResponse{Total: len(globalBoard.PlayerNames), Players: globalBoard.PlayerNames}
				playersMsg, _ := json.Marshal(&PlayerGone{Type: "bla", Players: ls})
				globalMutex.Unlock()

				log.Println("before 84")
				manager.send(playersMsg, conn)

				log.Println("before close(conn.send)")
				close(conn.send)
				log.Println("before delete(manager.clients, conn)")
				delete(manager.clients, conn)
				log.Println("before json.Marshal(&Message{Content")
				jsonMessage, _ := json.Marshal(&Message{Content: "/A socket has disconnected."})
				log.Println("before manager.send")
				manager.send(jsonMessage, conn)
			}
		case message := <-manager.broadcast:
			log.Println("send broadcast message")
			var msg Message
			json.Unmarshal(message, &msg)
			for conn := range manager.clients {
				log.Println("conn:" + conn.id)
				if msg.Content == "board" {
					if msg.Recipient != "" && msg.Recipient[0] != '^' && msg.Recipient != conn.id {
						continue
					}
					if msg.Recipient != "" && msg.Recipient[0] == '^' && msg.Recipient[1:] == conn.id {
						continue
					}
					gm := GetGameState(conn.id)
					jsonMessage, _ := json.Marshal(&gm)
					//log.Println(string(jsonMessage))

					log.Println("Going to send the following state to ", conn.id)
					var prettyJSON bytes.Buffer
					json.Indent(&prettyJSON, jsonMessage, "", "\t")
					//log.Println("Game State:", string(prettyJSON.Bytes()))

					message, _ = json.Marshal(&Message{Sender: msg.Sender, Content: string(jsonMessage)})
				}

				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(manager.clients, conn)
				}
			}
			log.Println("after iteration over conns")
		}
	}
}

func (c *Client) write() {
	defer func() {
		log.Println("client:", c.id, " write error. terminate thread")
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
		log.Println("client ", c.id, " read end (probably socket closed)")
		//c.socket.Close()
	}()
	log.Println("client read start")
	for {
		_, message, err := c.socket.ReadMessage()
		notifyAll := false
		if err != nil {
			log.Println("socket read failure")

			globalMutex.Lock()
			if (globalBoard.State == 0 || (globalBoard.State >= VictoryForGood && globalBoard.State <= VictoryForGawain)) && len(globalBoard.PlayerNames) > 0 {
				log.Println("no game yet so we can remove player from player list: ", globalBoard.PlayerNames)
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
		recipient := ""
		log.Println("successfully read message. client:", c.id, ". type: ", tp)
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
		} else if tp == "lady_suggest" {
			isGameCommand = true
			var sg LadySuggestMessage
			json.Unmarshal(message, &sg)
			LadySuggestHandler(sg.Content)
		} else if tp == "lady_response" {
			isGameCommand = true
			var sg LadyResponseMessage
			json.Unmarshal(message, &sg)
			LadyResponseHandler(sg.Content)
		} else if tp == "lady_publish_response" {
			isGameCommand = true
			var sg LadyPublishResponseMessage
			json.Unmarshal(message, &sg)
			LadyPublishResponseHandler(sg.Content)
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
			globalBoard = BoardGame{
				QuestStage:               1,
				lancelotCards:            make([]int, 7),
				PlayersWithBadCharacter:  make([]string, 0),
				playersWithGoodCharacter: make([]string, 0),
				playersWithCharacters: 	make(map[string]string),
				Secrets:                  make(map[string][]string),
				SecretsMap: map[string]*PlayerSecrets{},
				clientIdToPlayerName:     globalBoard.clientIdToPlayerName,
				manager:                  globalBoard.manager,
				PlayerToMurderInfo:       make(map[string]MurderInfo),
				PlayerNames:              globalBoard.PlayerNames,
				quests: QuestManager{
					current:                    0,
					playersVotes:               make([][]int, 20),
					results:                    make(map[int]QuestStats),
					realResults:                make(map[int]QuestStats),
					successfulQuest:            0,
					unsuccessfulQuest:          0,
					playerVotedForCurrent:      make(map[string]int),
					playerVotedForCurrentQuest: make([]string, 0),
					differentResults:           make(map[int]int),
					Flags:                      make(map[int]bool),
				},
			}

			globalMutex.Unlock()
		}
		if isGameCommand == true {

			if isOnlyForSender {
				recipient = c.id
			}
			if isOnlyForAllExceptSender {
				recipient = "^" + c.id
			}
			jsonMessage, _ := json.Marshal(&Message{Sender: c.id, Recipient: recipient, Content: "board"})
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
