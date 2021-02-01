package main

import (
	"log"
	"math"
	"strconv"
	"strings"
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
	suggestedPlayersString := strings.Join(suggestedPlayers[:], ",")
	globalBoard.StateDescription = "Vote For New Suggestion: by: " + globalBoard.PlayerNames[suggesterIn].Player + "; Suggested Players: " + suggestedPlayersString + "; Excalibur: " + pl.ExcaliburPlayer
	globalBoard.votesForNextMission = make(map[string]bool)
	globalBoard.suggestions.playersVotedYes = make([]string, 0)
	globalBoard.suggestions.playersVotedNo = make([]string, 0)

	/* Hammer */
	if globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].RetriesPerLevel[globalBoard.quests.current]-1 ==
		globalBoard.suggestions.unsuccessfulRetries {

		if HandleAcceptedSuggestion(globalBoard.numOfPlayers, newEntry) {
			return
		}

		allPlayers := make([]string, len(globalBoard.PlayerNames))
		for _, player := range globalBoard.PlayerNames {
			allPlayers = append(allPlayers, player.Player)
		}

		newEntry.PlayersVotedYes = allPlayers
		newEntry.LadySuggester = globalBoard.ladyOfTheLake.currentSuggester //lady of the lake
		globalBoard.suggestions.playersVotedYes = allPlayers

		globalBoard.suggestions.suggesterIndex = (globalBoard.suggestions.suggesterIndex+1) % len(globalBoard.PlayerNames)

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
	suggesterIn := globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	globalBoard.StateDescription = globalBoard.PlayerNames[suggesterIn].Player +
		" is suggesting " + globalBoard.suggestions.SuggestedTemporaryPlayers + "..."
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

	if len(globalBoard.votesForNextMission) == globalBoard.numOfPlayers { //last vote
		log.Println("vote is over. num of players =", len(globalBoard.PlayerNames))

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

			if HandleAcceptedSuggestion(numOfQuests, curEntry) {
				return
			}
		} else {
			globalBoard.State = WaitingForSuggestion

			suggesterIndex := globalBoard.suggestions.suggesterIndex
			globalBoard.StateDescription = "Suggestion For Next Quest: " + globalBoard.PlayerNames[suggesterIndex].Player +
				" is choosing players..."

			globalBoard.QuestStage += 0.1
			globalBoard.QuestStage = float32(math.Round(float64(globalBoard.QuestStage*100)) / 100)
			globalBoard.suggestions.unsuccessfulRetries++
		}

		globalBoard.isSuggestionGood, globalBoard.isSuggestionBad = 0, 0
		globalBoard.suggestions.suggesterIndex++
		globalBoard.suggestions.suggesterIndex = globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	}
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry

	globalMutex.Unlock()
}

func HandleAcceptedSuggestion(numOfQuests int, curEntry QuestArchiveItem) bool {
	/*
		Gawain's logic: If it's the last quest, the suggestion was accepted
		and Gawain is included - he WINS the game!
	*/
	if gawainPlayer, exists := isCharacterExists(true, "Gawain"); exists {
		if globalBoard.quests.current+1 == numOfQuests {
			for _, c := range globalBoard.suggestions.SuggestedPlayers {
				if c == gawainPlayer.Player {
					globalBoard.State = VictoryForGawain
					globalBoard.StateDescription = "VICTORY for Gawain"
					globalMutex.Unlock()
					return true
				}
			}
		}
	}

	isPellinoreInQuest := false
	isBeastInQuest := false
	for _, c := range globalBoard.suggestions.SuggestedPlayers {
		if c == Pellinore {
			isPellinoreInQuest = true
		}
		if c == TheQuestingBeast {
			isBeastInQuest = true
		}
	}
	if isPellinoreInQuest && isBeastInQuest {
		globalBoard.quests.Flags[BEAST_AND_PELLINORE_AT_SAME_QUEST] = true
	}
	globalBoard.State = JorneyVoting
	globalBoard.StateDescription = "The Quest was accepted. The Vote for Quest " + strconv.Itoa(globalBoard.quests.current+1) + " is starting now... "
	curEntry.IsSuggestionAccepted = true
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
	return false
}

func UncoverSuggesterToViviana() {
	suggesterIndex := globalBoard.suggestions.suggesterIndex % len(globalBoard.PlayerNames)
	suggesterPlayerName := globalBoard.PlayerNames[suggesterIndex]
	suggesterCharacter := globalBoard.PlayerToCharacter[suggesterPlayerName]
	strayPlayer, strayExists := isCharacterExists(true, Stray)
	if strayExists && strayPlayer == suggesterPlayerName {
		suggesterCharacter = Stray
	}
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