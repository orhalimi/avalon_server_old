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

		if SirKay == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
		} else if Mordred == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
		} else if Lot == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
			vivianaSecrets := make([]string, 1)
			vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player+" is Lot")
			globalBoard.Secrets[Viviana] = vivianaSecrets
		} else if "Gawain" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
			vivianaSecrets := globalBoard.Secrets[Viviana]
			if vivianaSecrets == nil {
				vivianaSecrets = make([]string, 1)
			}
			vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player+" is Gawain")
			globalBoard.Secrets[Viviana] = vivianaSecrets
		} else if "Ginerva" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
			vivianaSecrets := globalBoard.Secrets[Viviana]
			if vivianaSecrets == nil {
				vivianaSecrets = make([]string, 1)
			}
			vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player+" is Gawain")
			globalBoard.Secrets[Viviana] = vivianaSecrets
		} else if _, isSuggesterBadCharacter := badCharacters[globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]]]; isSuggesterBadCharacter {
			log.Println("suggester is bad")
			globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
		} else {
			log.Println("suggester is good")
			globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
		}

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
			curEntry.IsSuggestionAccepted = true
			globalBoard.suggestions.LastUnsuccessfulRetries = globalBoard.suggestions.unsuccessfulRetries
			globalBoard.suggestions.unsuccessfulRetries = 0
			globalBoard.LastQuestStage = globalBoard.QuestStage
			globalBoard.QuestStage += 0.01
			globalBoard.QuestStage = float32(math.Ceil(float64(globalBoard.QuestStage)))

			//for vivian
			if SirKay == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
			} else if Mordred == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
			} else if Lot == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
				vivianaSecrets := globalBoard.Secrets[Viviana]
				if vivianaSecrets == nil {
					vivianaSecrets = make([]string, 1)
				}
				if nil == globalBoard.playersWithCharacters {
					globalBoard.playersWithCharacters = make(map[string]string)
				}
				globalBoard.playersWithCharacters[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player] = Lot
				vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player+" is Lot")
				globalBoard.Secrets[Viviana] = vivianaSecrets
			} else if "Gawain" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
				vivianaSecrets := globalBoard.Secrets[Viviana]
				if vivianaSecrets == nil {
					vivianaSecrets = make([]string, 1)
				}
				if nil == globalBoard.playersWithCharacters {
					globalBoard.playersWithCharacters = make(map[string]string)
				}
				globalBoard.playersWithCharacters[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player] = "Gawain"
				vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player+" is Gawain")
				globalBoard.Secrets[Viviana] = vivianaSecrets
			} else if "Ginerva" == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
				vivianaSecrets := globalBoard.Secrets[Viviana]
				if vivianaSecrets == nil {
					vivianaSecrets = make([]string, 1)
				}
				vivianaSecrets = append(vivianaSecrets, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player+" is Ginerva")
				globalBoard.Secrets[Viviana] = vivianaSecrets
			} else if Stray == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				if nil == globalBoard.playersWithCharacters {
					globalBoard.playersWithCharacters = make(map[string]string)
				}
				globalBoard.playersWithCharacters[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player] = Stray
			} else if Oberon == globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]] {
				if nil == globalBoard.playersWithCharacters {
					globalBoard.playersWithCharacters = make(map[string]string)
				}
				globalBoard.playersWithCharacters[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player] = Oberon
			} else if _, isSuggesterBadCharacter := badCharacters[globalBoard.PlayerToCharacter[globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)]]]; isSuggesterBadCharacter {
				log.Println("suggester is bad")
				globalBoard.playersWithBadCharacter = append(globalBoard.playersWithBadCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
			} else {
				log.Println("suggester is good")
				globalBoard.playersWithGoodCharacter = append(globalBoard.playersWithGoodCharacter, globalBoard.PlayerNames[globalBoard.suggestions.suggesterIndex%len(globalBoard.PlayerNames)].Player)
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
