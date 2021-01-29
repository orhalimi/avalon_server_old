package main

import "log"

type SecretResponse struct {
	Character                string   `json:"character,omitempty"`
	Secrets                  []string `json:"secrets,omitempty"`
	PlayersWithGoodCharacter []string `json:"goodplayers,omitempty"` //for vivian
	PlayersWithBadCharacter  []string `json:"badplayers,omitempty"`  //for vivian
	PlayersWithCharacters  map[string]string `json:"uncoveredplayers,omitempty"`  //for vivian
}

func GetNightSecretsFromPlayerName(player PlayerName) SecretResponse {
	response := SecretResponse{}

	if player.Player == "" {
		return SecretResponse{}
	}

	character := globalBoard.PlayerToCharacter[player]
	response.Character = character
	response.Secrets = globalBoard.Secrets[player.Player]

	if character == Viviana {
		log.Println(globalBoard.playersWithBadCharacter)
		log.Println(globalBoard.playersWithGoodCharacter)
		response.PlayersWithBadCharacter = globalBoard.playersWithBadCharacter
		response.PlayersWithGoodCharacter = globalBoard.playersWithGoodCharacter
		response.PlayersWithCharacters = globalBoard.playersWithCharacters
		response.Secrets = globalBoard.Secrets[player.Player]
	}
	return response
}


func getTypeOfLevel(levelNum int, numOfPlayers int) int {
	log.Println(levelNum, numOfPlayers)
	if numOfPlayers <= 6 {
		return RegularQuest
	}
	if levelNum == 4 {
		return FlushQuest
	} else if globalConfigPerNumOfPlayers[numOfPlayers].NumOfQuests-1 == levelNum {
		return TwoFailsRequiredQuest
	}
	return RegularQuest

}

type PlayerInfo struct {
	Character  string `json:"ch,omitempty"`
	IsKilled   bool   `json:"isKilled,omitempty"`
	isKilledBy []string
}


type GameState struct {
	Players                   ListOfPlayersResponse           `json:"players,omitempty"`
	CurrentQuest              int                             `json:"current,omitempty"`
	NumOfActivePlayers        int                        `json:"active_players_num,omitempty"`
	Characters                map[string]CharacterDescription `json:"characters,omitempty"`
	Size                      int                             `json:"size,omitempty"`
	State                     int                             `json:"state"`
	StateDescription          string                            `json:"stateDescription"`
	Archive                   []QuestArchiveItem              `json:"archive"`
	Secrets                   SecretResponse                  `json:"secrets"`
	Suggester                 string                          `json:"suggester,omitempty"`
	Murder                    Murder                          `json:"murder,omitempty"`

	/* Seer's two options to see: the player before and player after him.
	The "Pick" field is unused in this state.
	*/
	SirPick                   SirPick                         `json:"sir,omitempty"`

	OptionalVotes             []string                        `json:"optionalVotes,omitempty"`
	SuggesterVeto             string                          `json:"suggesterVeto,omitempty"`
	OnlyGoodSuggested         bool                        		`json:"onlyGoodSuggested,omitempty"`
	SuggestedPlayers          []string                        `json:"suggestedPlayers,omitempty"`
	SuggestedTemporaryPlayers string                          `json:"suggestedTemporaryPlayers,omitempty"`
	PlayersVotedForCurrQuest  []string                        `json:"PlayersVotedForCurrQuest,omitempty"`
	PlayersVotedYes           []string                        `json:"PlayersVotedYesForSuggestion,omitempty"`
	PlayersVotedNo            []string                        `json:"PlayersVotedNoForSuggestion,omitempty"`
	Results                   map[int]QuestStats              `json:"results,omitempty"`
	PlayerInfo                map[string]PlayerInfo           `json:"playerToCharacters,omitempty"`
	IsExcalibur               bool                            `json:"excalibur,omitempty"`
	SuggestedExcalibur        string                          `json:"suggestedExcalibur,omitempty"`
	IsLady                    bool                            `json:"isLady,omitempty"`              //lady of the lake
	LadySuggester             string                          `json:"ladySuggester,omitempty"`       //lady of the lake
	LadyChosenPlayer          string                          `json:"ladyChosenPlayer,omitempty"`    //lady of the lake
	LadyResponse              string                          `json:"ladyResponse,omitempty"`        //lady of the lake
	LadyResponseOptions       []string                        `json:"ladyResponseOptions,omitempty"` //lady of the lake
	LadyPublish               string                          `json:"ladyPublish,omitempty"`         //lady of the lake
	LadyPreviousSuggester               string                `json:"ladyPreviousSuggester,omitempty"`     //lady of the lake
}

func GetGameState(clientId string) GameState {
	globalMutex.RLock()
	board := GameState{}

	// Seer's two options to see: the player before or after.
	if globalBoard.State == SirPickPlayer && globalBoard.CharacterToPlayer[Seer].Player == clientId {
		seerOptions := getSeerOptions(clientId)
		board.SirPick = SirPick{Options: seerOptions}
	}

	if globalBoard.quests.Flags[EXCALIBUR] {
		board.IsExcalibur = true
		board.SuggestedExcalibur = globalBoard.suggestions.excalibur.Player
	}
	if globalBoard.quests.Flags[LADY] {
		board.IsLady = true
		board.LadySuggester = globalBoard.ladyOfTheLake.currentSuggester
		board.LadyChosenPlayer = globalBoard.ladyOfTheLake.currentChosenPlayer
		if globalBoard.ladyOfTheLake.currentSuggester == clientId && globalBoard.State == LadySuggesterPublishResponseToWorld {
			log.Println("response: ", globalBoard.ladyOfTheLake.ladyResponse)
			if globalBoard.ladyOfTheLake.ladyResponse == 0 {
				board.LadyResponse = "Bad"
			} else if globalBoard.ladyOfTheLake.ladyResponse == 1 {
				board.LadyResponse = "Good"
			}
		} else {
			log.Println(globalBoard.ladyOfTheLake.currentSuggester, clientId)
		}
		board.LadyResponseOptions = getOptionalLoyalty(clientId)
		board.LadyPreviousSuggester = globalBoard.ladyOfTheLake.previousSuggester
	}
	board.SuggestedTemporaryPlayers = globalBoard.suggestions.SuggestedTemporaryPlayers
	board.Players.Total = len(globalBoard.PlayerNames)
	board.Players.Players = globalBoard.PlayerNames
	players := make([]PlayerName, 0)
	for _, p := range globalBoard.PlayerNames {
		if globalBoard.PlayerToCharacter[p] != Ector {
			players = append(players, p)
		}
	}
	board.Players.Active = players
	board.SuggestedPlayers = globalBoard.suggestions.SuggestedPlayers
	board.CurrentQuest = globalBoard.quests.current + 1
	board.NumOfActivePlayers = globalBoard.numOfPlayers
	board.CurrentQuest = globalBoard.quests.current + 1
	board.Size = globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].NumOfQuests
	board.Results = globalBoard.quests.results
	board.Characters = make(map[string]CharacterDescription)
	str, ok := globalBoard.CharacterToPlayer[Stray]
	strayNewCharacter := globalBoard.PlayerToCharacter[str]
	for _, ch := range globalBoard.Characters {
		if ok && Stray == ch && str.Player == clientId {
			board.Characters[ch] = CharactersDescriptionMap[strayNewCharacter]
		} else {
			board.Characters[ch] = CharactersDescriptionMap[ch]
		}
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

	if clientId == Meliagant {
		board.OnlyGoodSuggested = globalBoard.suggestions.OnlyGoodSuggested
	}
	board.State = globalBoard.State
	board.StateDescription = globalBoard.StateDescription
	board.Secrets = GetNightSecretsFromPlayerName(PlayerName{clientId})
	board.OptionalVotes = getOptionalVotesAccordingToQuestMembers(globalBoard.PlayerToCharacter[PlayerName{clientId}], globalBoard.suggestions.SuggestedCharacters, globalBoard.quests.Flags, globalBoard.quests.current, globalBoard.numOfPlayers)
	board.PlayersVotedYes = globalBoard.suggestions.playersVotedYes
	board.PlayersVotedNo = globalBoard.suggestions.playersVotedNo
	if len(globalBoard.PlayerNames) > 0 {
		board.Suggester = globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player
	}
	if globalBoard.State == MurdersAfterBadVictory || globalBoard.State == MurdersAfterGoodVictory {
		board.Murder.TargetCharacters = globalBoard.PendingMurders[0].TargetCharacters
		board.Murder.By = globalBoard.PendingMurders[0].By
		board.Murder.ByCharacter = globalBoard.PendingMurders[0].ByCharacter
	}

	//Here we can expose character in the player list!!!

	if globalBoard.PlayerToCharacter[PlayerName{clientId}] == Viviana {
		board.PlayerInfo = make(map[string]PlayerInfo)
		for p, c := range globalBoard.playersWithCharacters {
			playerInfo := PlayerInfo{}
			playerInfo.Character = c
			board.PlayerInfo[p] = playerInfo
		}
	}

	ectorName, hasEctor := globalBoard.CharacterToPlayer[Ector]
	if hasEctor || globalBoard.State == VictoryForGood || globalBoard.State == VictoryForBad ||
		globalBoard.State == VictoryForSirGawain {
		board.PlayerInfo = make(map[string]PlayerInfo)
		if globalBoard.State == VictoryForSirGawain || globalBoard.State == VictoryForGood || globalBoard.State == VictoryForBad {
			for _, pl := range board.Players.Players {
				playerInfo := PlayerInfo{}
				playerInfo.Character = globalBoard.PlayerToCharacter[pl]
				_, isKilled := globalBoard.PlayerToMurderInfo[pl.Player]
				playerInfo.IsKilled = isKilled
				board.PlayerInfo[pl.Player] = playerInfo
			}
		}
		if hasEctor {
			playerInfo := PlayerInfo{Character: Ector, IsKilled: false}
			board.PlayerInfo[ectorName.Player] = playerInfo
		}
	}
	if globalBoard.State == MurdersAfterBadVictory || globalBoard.State == MurdersAfterGoodVictory {
		dagonetName, hasDagonet := globalBoard.CharacterToPlayer[Dagonet]
		if hasDagonet {
			if board.PlayerInfo == nil {
				board.PlayerInfo = make(map[string]PlayerInfo)
			}
			playerInfo := PlayerInfo{Character: Dagonet, IsKilled: false}
			board.PlayerInfo[dagonetName.Player] = playerInfo
		}
	}
	globalMutex.RUnlock()
	return board
}

func getSeerOptions(clientId string) []string {
	var seerIndex int
	for i, p := range globalBoard.PlayerNames {
		if p.Player == clientId {
			seerIndex = i
		}
	}

	indexBeforeSeer := seerIndex - 1
	if indexBeforeSeer < 0 {
		indexBeforeSeer = len(globalBoard.PlayerNames) - 1
	}

	indexAfterSeer := seerIndex + 1
	if indexAfterSeer > len(globalBoard.PlayerNames)-1 {
		indexAfterSeer = 0
	}
	seerOptions := []string{globalBoard.PlayerNames[indexBeforeSeer].Player,
		globalBoard.PlayerNames[indexAfterSeer].Player}
	return seerOptions
}
