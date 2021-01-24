package main

import (
	"log"
	"math"
)

type Suggestion struct {
	Players         []string `json:"players,omitempty"`
	ExcaliburPlayer string   `json:"excalibur,omitempty"`
}


type SuggestMessage struct {
	Tp      string     `json:"type"`
	Content Suggestion `json:"content"`
}


type SuggestTmpMessage struct {
	Tp      string   `json:"type"`
	Content []string `json:"content"`
}

type VoteForSuggestionMessage struct {
	Tp      string            `json:"type"`
	Content VoteForSuggestion `json:"content"`
}


type VoteForSuggestion struct {
	PlayerName string `json:"playerName"`
	Vote       bool   `json:"vote"`
}

func HandleNewSuggest(pl Suggestion) {
	globalMutex.Lock()
	suggestedPlayers := pl.Players
	suggestedCharacters := make(map[string]bool, 0)

	for _, v := range pl.Players {
		suggestedCharacters[globalBoard.PlayerToCharacter[PlayerName{v}]] = true
	}

	suggesterIn := globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	newEntry := QuestArchiveItem{Id: globalBoard.QuestStage, Suggester: globalBoard.PlayerNames[suggesterIn], SuggestedPlayers: suggestedPlayers, ExcaliburPlayer: pl.ExcaliburPlayer}

	log.Println("SuggestedPlayers:", suggestedPlayers, ",ExcaliburPlayer:", pl.ExcaliburPlayer, ",Suggester:", globalBoard.PlayerNames[suggesterIn].Player)
	globalBoard.suggestions.SuggestedTemporaryPlayers = ""
	globalBoard.suggestions.SuggestedPlayers = suggestedPlayers
	globalBoard.suggestions.SuggestedCharacters = suggestedCharacters
	globalBoard.suggestions.excalibur.Player = pl.ExcaliburPlayer
	globalBoard.suggestions.excalibur.Suggester = globalBoard.PlayerNames[suggesterIn].Player
	globalBoard.suggestions.OnlyGoodSuggested = false

	allGood := true
	for ch, val := range suggestedCharacters {
		if val && badCharacters[ch] {
			allGood = false
			break
		}
	}
	if _, ok := suggestedCharacters["Gawain"]; ok {
		allGood = false
	}
	if _, ok := suggestedCharacters["Ginerva"]; ok {
		allGood = false
	}
	if _, ok := suggestedCharacters[Lot]; ok {
		allGood = false
	}
	if allGood {
		globalBoard.suggestions.OnlyGoodSuggested = true
	}

	globalBoard.State = SuggestionVoting
	globalBoard.votesForNextMission = make(map[string]bool)
	globalBoard.suggestions.playersVotedYes = make([]string, 0)
	globalBoard.suggestions.playersVotedNo = make([]string, 0)

	if globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].RetriesPerLevel[globalBoard.quests.current]-1 == globalBoard.suggestions.unsuccessfulRetries {
		globalBoard.State = JorneyVoting
		globalBoard.suggestions.unsuccessfulRetries = 0

		allPlayers := make([]string, len(globalBoard.PlayerNames))
		for _, player := range globalBoard.PlayerNames {
			allPlayers = append(allPlayers, player.Player)
		}

		UncoverSuggesterToViviana()

		newEntry.IsSuggestionAccepted = true
		newEntry.IsSuggestionOver = true
		newEntry.PlayersVotedYes = allPlayers
		newEntry.LadySuggester = globalBoard.ladyOfTheLake.currentSuggester //lady of the lake
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


func isCharacterExists(lockHeld bool, character string) (PlayerName, bool) {
	if !lockHeld {
		globalMutex.RLock()
	}

	player, exists := globalBoard.CharacterToPlayer[character]

	if !lockHeld {
		globalMutex.RUnlock()
	}
	return player, exists
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

		numOfQuests := globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].NumOfQuests
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

			/*
				Gawain's logic: If it's the last quest, the suggestion was accepted
				and Gawain is included - he WINS the game!
			*/
			if gawainPlayer, exists := isCharacterExists(true, "Gawain"); exists {
				if globalBoard.quests.current + 1 == numOfQuests {
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
			curEntry.IsSuggestionAccepted = true
			globalBoard.suggestions.LastUnsuccessfulRetries = globalBoard.suggestions.unsuccessfulRetries
			globalBoard.suggestions.unsuccessfulRetries = 0
			globalBoard.LastQuestStage = globalBoard.QuestStage
			globalBoard.QuestStage += 0.01
			globalBoard.QuestStage = float32(math.Ceil(float64(globalBoard.QuestStage)))

			if _, exists := isCharacterExists(true, Viviana); exists {
				/* Suggestion was accepted, viviana get a new secret about the suggester. */
				UncoverSuggesterToViviana()
			}

			if globalBoard.quests.Flags[HAS_BALAIN_AND_BALIN] {
				balinPlayer := globalBoard.CharacterToPlayer[Balin]
				balainPlayer := globalBoard.CharacterToPlayer[Balain]
				balinIsSuggestion := false
				balainIsSuggestion := false
				for _, c := range globalBoard.suggestions.SuggestedPlayers {
					if c == balinPlayer.Player {
						balinIsSuggestion = true
					}
					if c == balainPlayer.Player {
						balainIsSuggestion = true
					}
				}
				if balainIsSuggestion && balinIsSuggestion {
					globalBoard.CharacterToPlayer[Balain] = balinPlayer
					globalBoard.CharacterToPlayer[Balin] = balainPlayer
					globalBoard.PlayerToCharacter[balinPlayer] = Balain
					globalBoard.PlayerToCharacter[balainPlayer] = Balin
				}

			}

			numOfUnsuccesfulRetries := globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].RetriesPerLevel[globalBoard.quests.current]
			suggesterVetoIn := (globalBoard.suggestions.suggesterIndex + 1 + numOfUnsuccesfulRetries) % len(globalBoard.PlayerNames)
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

func UncoverSuggesterToViviana() {
	suggesterIndex := globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	suggesterPlayerName := globalBoard.PlayerNames[suggesterIndex]
	suggesterCharacter := globalBoard.PlayerToCharacter[suggesterPlayerName]

	if SirKay == suggesterCharacter {
		UncoverAsBadCharacter(suggesterPlayerName)
	} else if Mordred == suggesterCharacter {
		UncoverAsGoodCharacter(suggesterPlayerName)
	} else if Lot == suggesterCharacter {
		UncoverAsBadCharacter(suggesterPlayerName)
		UncoverCharacter(suggesterPlayerName, Lot)
	} else if "Gawain" == suggesterCharacter {
		UncoverAsBadCharacter(suggesterPlayerName)
		UncoverCharacter(suggesterPlayerName, "Gawain")
	} else if "Ginerva" == suggesterCharacter {
		UncoverAsBadCharacter(suggesterPlayerName)
	} else if Stray == suggesterCharacter {
		UncoverCharacter(suggesterPlayerName, Stray)
	} else if Oberon == suggesterCharacter {
		UncoverCharacter(suggesterPlayerName, Oberon)
	} else if _, isSuggesterBadCharacter := badCharacters[suggesterCharacter]; isSuggesterBadCharacter {
		log.Println("suggester is bad")
		UncoverAsBadCharacter(suggesterPlayerName)
	} else {
		log.Println("suggester is good")
		UncoverAsGoodCharacter(suggesterPlayerName)
	}
}

func UncoverCharacter(suggesterPlayerName PlayerName, character string) {
	if nil == globalBoard.playersWithCharacters {
		globalBoard.playersWithCharacters = make(map[string]string)
	}
	globalBoard.playersWithCharacters[suggesterPlayerName.Player] = character
}

func UncoverAsBadCharacter(suggesterPlayerName PlayerName) {
	globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter,
		suggesterPlayerName.Player)
}

func UncoverAsGoodCharacter(suggesterPlayerName PlayerName) {
	globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter,
		suggesterPlayerName.Player)
}