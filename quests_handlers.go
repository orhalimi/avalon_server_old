package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

const (
	// Possible votes
	VoteFail        = 0
	VoteSuccess     = 1
	VoteReversal    = 2
	VoteBeast       = 3
	VoteAvalonPower = 5
)

type VoteForJourney struct {
	PlayerName string `json:"playerName,omitempty"`
	Vote       int    `json:"vote,omitempty"`
}

type VoteForJourneyMessage struct {
	Tp      string         `json:"type"`
	Content VoteForJourney `json:"content"`
}

func HandleJourneyVote(vote VoteForJourney) {
	globalMutex.Lock()
	current := globalBoard.quests.current

	if globalBoard.State != JorneyVoting {
		return
	}

	if _, ok := globalBoard.quests.playerVotedForCurrent[vote.PlayerName]; ok {
		globalMutex.Unlock()
		return
	}

	if globalBoard.PlayerToCharacter[PlayerName{vote.PlayerName}] == Titanya &&
		vote.Vote == VoteFail {
		globalBoard.quests.Flags[TITANYA_FIRST_FAIL] = true
	}

	if globalBoard.PlayerToCharacter[PlayerName{vote.PlayerName}] == Elaine &&
		vote.Vote == VoteAvalonPower {
		globalBoard.quests.Flags[ELAINE_AVALON_POWER_CARD] = true
	}

	if globalBoard.PlayerToCharacter[PlayerName{vote.PlayerName}] == TheQuestingBeast &&
		vote.Vote == VoteSuccess {
		globalBoard.quests.Flags[BEAST_FIRST_SUCCESS] = true
	}

	origVote := vote.Vote
	if origVote == VoteBeast {
		globalBoard.quests.Flags[BEAST_VOTE_SEEN] = true
		origVote = VoteFail
	}
	log.Println(vote.PlayerName, " voted ", vote.Vote)

	globalBoard.quests.playerVotedForCurrentQuest = append(globalBoard.quests.playerVotedForCurrentQuest, vote.PlayerName)

	votedPlayersString := strings.Join(globalBoard.quests.playerVotedForCurrentQuest[:], ",")
	globalBoard.StateDescription = " Voting for Quest " + strconv.Itoa(current+1) + "!" + votedPlayersString + " voted!"

	globalBoard.quests.playerVotedForCurrent[vote.PlayerName] = origVote
	mp := append(globalBoard.quests.playersVotes[current], origVote)

	res := globalBoard.quests.results[current+1]
	requiredVotes := res.NumOfPlayers

	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	if vote.Vote == VoteFail {
		res.NumOfFailures++
		curEntry.NumberOfFailures++
	} else if vote.Vote == VoteSuccess {
		res.NumOfSuccess++
		curEntry.NumberOfSuccesses++
	} else if vote.Vote == VoteBeast {
		res.NumOfBeasts++
		curEntry.NumberOfBeasts++
	} else if vote.Vote == VoteReversal {
		res.NumOfReversal++
		curEntry.NumberOfReversal++
	}


	if len(mp) == requiredVotes { //last vote
		if _, ok := globalBoard.quests.Flags[EXCALIBUR]; ok {
			globalBoard.State = ExcaliburPick
			//update info
			globalBoard.StateDescription = "Excalibur: " + globalBoard.suggestions.excalibur.Player +
				" is deciding whether to reverse some vote or not..."
			globalBoard.archive[len(globalBoard.archive)-1] = curEntry
			globalBoard.quests.results[current+1] = res
			globalBoard.quests.playersVotes[current] = mp
			globalMutex.Unlock()
			return
		}

		if StartNewSuggestion(mp, curEntry, current) {
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

func StartNewSuggestion(mp []int, curEntry QuestArchiveItem, current int) bool {
	for _, vote := range mp {
		if vote == VoteAvalonPower {
			globalBoard.State = WaitingForSuggestion
			suggesterIndex := globalBoard.suggestions.suggesterIndex
			globalBoard.StateDescription = "Suggestion For Next Quest: " + globalBoard.PlayerNames[suggesterIndex].Player +
				" is choosing players..."

			curEntry.AvalonPower = true

			playerWithVeto := globalBoard.suggestions.PlayerWithVeto
			vetoIndex := 0
			for i, p := range globalBoard.PlayerNames {
				if playerWithVeto == p.Player {
					vetoIndex = i + 1
					break
				}
			}
			vetoIndex = vetoIndex % len(globalBoard.PlayerNames)
			globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[vetoIndex].Player

			globalBoard.quests.playerVotedForCurrentQuest = make([]string, 0)
			globalBoard.quests.playerVotedForCurrent = make(map[string]int)
			globalBoard.votesForNextMission = make(map[string]bool) //for suggestions
			globalBoard.suggestions.SuggestedPlayers = make([]string, 0)
			globalBoard.quests.playersVotes[current] = make([]int, 0)
			globalBoard.archive[len(globalBoard.archive)-1] = curEntry

			globalBoard.QuestStage = globalBoard.LastQuestStage

			globalMutex.Unlock()
			return true
		}
	}
	return false
}

func EndJourney(res *QuestStats, mp []int, curEntry *QuestArchiveItem, current int) {
	retriesPerLevel := globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].RetriesPerLevel
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
	if playerName, ok := globalBoard.CharacterToPlayer[KingArthur]; ok {
		//King-Arthur is playing
		if vote, ok := globalBoard.quests.playerVotedForCurrent[playerName.Player]; ok {
			//King-Arthur was in this quest
			log.Println("switch King-Arthur's \"Fail\" to \"Success")
			realResults := make([]int, len(mp))
			copy(realResults, mp)
			for i, res := range realResults {
				if res == vote {
					realResults[i] = (1 + vote) % 2
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
	numOfExpectedQuests := globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].NumOfQuests
	if globalBoard.quests.successfulQuest > numOfExpectedQuests/2 {
		pendingMurders, hasMurders := GetMurdersAfterGoodsWins()
		if !hasMurders {
			globalBoard.State = VictoryForGood
			globalBoard.StateDescription = "VICTORY for Goods"
		} else {
			fmt.Println(pendingMurders)
			globalBoard.State = MurdersAfterGoodVictory
			globalBoard.PendingMurders = pendingMurders

			targetCharactersString := strings.Join(globalBoard.PendingMurders[0].TargetCharacters[:], ",")
			globalBoard.StateDescription = "Murder: " + globalBoard.PendingMurders[0].ByCharacter + " is trying to kill: " +
				targetCharactersString
		}
	} else if isBadVictory(globalBoard.quests.unsuccessfulQuest, numOfExpectedQuests) {
		pendingMurders, hasMurders := GetMurdersAfterBadsWins()
		if !hasMurders {
			globalBoard.State = VictoryForBad
			globalBoard.StateDescription = "VICTORY for Bads"
		} else {
			globalBoard.State = MurdersAfterBadVictory
			globalBoard.PendingMurders = pendingMurders

			targetCharactersString := strings.Join(globalBoard.PendingMurders[0].TargetCharacters[:], ",")
			globalBoard.StateDescription = "Murders: " + globalBoard.PendingMurders[0].ByCharacter + " should kill: " +
				targetCharactersString
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
					lanBad := globalBoard.CharacterToPlayer[LancelotBad]
					lanGood := globalBoard.CharacterToPlayer[LancelotGood]
					globalBoard.CharacterToPlayer[LancelotBad] = lanGood
					globalBoard.CharacterToPlayer[LancelotGood] = lanBad
					globalBoard.PlayerToCharacter[lanBad] = LancelotGood
					globalBoard.PlayerToCharacter[lanGood] = LancelotBad
					curEntry.IsSwitchLancelot = true
					//fix bug of viviana that seeother lanselot
					/*for i, pl := range globalBoard.PlayersWithBadCharacter {
						if pl == lanBad.Player {
							globalBoard.PlayersWithBadCharacter[i] = lanGood.Player
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
					lanBad := globalBoard.CharacterToPlayer[LancelotBad]
					globalBoard.CharacterToPlayer[LancelotGood] = lanBad
					globalBoard.PlayerToCharacter[lanBad] = LancelotGood
					delete(globalBoard.CharacterToPlayer, LancelotBad)
					for i, ch := range globalBoard.Characters {
						if ch == LancelotBad {
							globalBoard.Characters = append(globalBoard.Characters[:i], globalBoard.Characters[i+1:]...)
						}
						globalBoard.Characters = append(globalBoard.Characters, LancelotGood)
						break
					}
					curEntry.IsSwitchLancelot = true
					delete(globalBoard.quests.Flags, HAS_ONLY_BAD_LANCELOT)
					globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] = true
				} else if globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] {
					lanBad := globalBoard.CharacterToPlayer[LancelotGood]
					globalBoard.CharacterToPlayer[LancelotBad] = lanBad
					globalBoard.PlayerToCharacter[lanBad] = LancelotBad
					delete(globalBoard.CharacterToPlayer, LancelotGood)
					for i, ch := range globalBoard.Characters {
						if ch == LancelotGood {
							globalBoard.Characters = append(globalBoard.Characters[:i], globalBoard.Characters[i+1:]...)
						}
						globalBoard.Characters = append(globalBoard.Characters, LancelotBad)
						break
					}
					curEntry.IsSwitchLancelot = true
					delete(globalBoard.quests.Flags, HAS_ONLY_GOOD_LANCELOT)
					globalBoard.quests.Flags[HAS_ONLY_BAD_LANCELOT] = true
				}
			}
		}
		//end of special actions after quest
		if globalBoard.quests.Flags[LADY] && globalBoard.quests.current >= 1 {
			globalBoard.State = WaitingForLadySuggester
			globalBoard.StateDescription = "Lady Of The Lake: " + globalBoard.ladyOfTheLake.currentSuggester +
				" is choosing player..."
		} else {
			globalBoard.State = WaitingForSuggestion
			suggesterIndex := globalBoard.suggestions.suggesterIndex
			globalBoard.StateDescription = "Suggestion For Next Quest: " + globalBoard.PlayerNames[suggesterIndex].Player +
			" is choosing players..."
		}

	}
	globalBoard.quests.playerVotedForCurrentQuest = make([]string, 0)
	globalBoard.quests.playerVotedForCurrent = make(map[string]int)
	globalBoard.votesForNextMission = make(map[string]bool) //for suggestions
	globalBoard.suggestions.SuggestedPlayers = make([]string, 0)
	globalBoard.suggestions.OnlyGoodSuggested = false
}

func isBadVictory(numOfUnsuccessfulQuests, numOfExpectedQuests int) bool {
	return numOfUnsuccessfulQuests > numOfExpectedQuests/2 || (numOfExpectedQuests == 4 && numOfUnsuccessfulQuests == 2)
}

func CalculateQuestResult(mp []int) int {
	result := JorneySuccess
	log.Println("++ last")
	NumOfFailures := 0
	NumOfReverse := 0
	for _, v := range mp {
		if v == VoteFail {
			NumOfFailures++
		}
		if v == VoteReversal {
			NumOfReverse++
		}
	}

	questType := getTypeOfLevel(globalBoard.quests.current+1, globalBoard.numOfPlayers)
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

func getOptionalVotesAccordingToQuestMembers(character string, questMembers map[string]bool,
	flags map[int]bool, current int, numOfPlayers int) []string {

	if character == "Gawain" {
		return []string{"Fail", "Success"}
	}

	if character == KingArthur {
		return []string{"Fail"}
	}
	if character == LancelotBad {
		return []string{"Fail"}
	}
	if character == Balin {
		return []string{"Fail"}
	}

	/*
		Titanya's optional votes: If it's the first vote for Titanya - she must vote "Fail".
		Afterward, she vote Success. If the bads needs one more failure for victory - Titanya
		votes Success!
	*/
	if character == Titanya {
		numOfExpectedQuests := globalConfigPerNumOfPlayers[numOfPlayers].NumOfQuests
		if isBadVictory(globalBoard.quests.unsuccessfulQuest+1, numOfExpectedQuests) {
			return []string{"Success"}
		}
		if _, ok := flags[TITANYA_FIRST_FAIL]; !ok {
			log.Println("titanya has fail")
			return []string{"Fail"}
		}
	}

	if character == Elaine {
		numOfExpectedQuests := globalConfigPerNumOfPlayers[numOfPlayers].NumOfQuests
		if _, ok := flags[ELAINE_AVALON_POWER_CARD]; !ok && numOfExpectedQuests != globalBoard.quests.current+1 {
			log.Println("elaine avalon card or success")
			return []string{"Success", "Avalon Power"}
		}
	}

	if character == Polygraph {
		return []string{"Fail"}
	}
	if character == Agravain {
		return []string{"Success"}
	}
	if character == Lot {
		return []string{"Success"}
	}
	if character == Nimue {
		if _, ok := questMembers[Merlin]; ok {
			if _, ok := questMembers[Galahad]; !ok {
				log.Println("nimue  has fail")
				return []string{"Fail"}
			}
		}
	}

	if FlushQuest == getTypeOfLevel(current+1, numOfPlayers) {
		if _, ok := badCharacters[character]; ok || character == "Ginerva" || character == Meliagant {
			return []string{"Fail"}
		} else {
			return []string{"Success"}
		}
	}

	if character == Meliagant {
		return []string{"Fail", "Success"}
	}

	if character == TheQuestingBeast {
		if _, ok := flags[BEAST_FIRST_SUCCESS]; !ok {
			return []string{"Success", "Beast"}
		} else {
			return []string{"Beast"}
		}
	}

	res := make([]string, 0)
	if character == BadAngel || character == GoodAngel {
		res = append(res, "Reversal")
	}

	res = append(res, "Success")

	p, ok := globalBoard.CharacterToPlayer[character]
	var isStray bool
	if ok {
		char, _ := globalBoard.PlayerToCharacter[p]
		if char == Stray {
			isStray = true
		}
	}
	if badCharacters[character] || character == "Puck" || character == "Ginerva" || isStray {
		res = append(res, "Fail")
	}
	log.Println(character, " has", res)
	return res
}

func getVoteStr(vote int) string {
	if VoteFail == vote {
		return "Fail"
	}
	if VoteSuccess == vote {
		return "Success"
	}
	if VoteReversal == vote {
		return "Reversal"
	}
	if VoteBeast == vote {
		return "Beast"
	}
	if VoteAvalonPower == vote {
		return "Avalon Power"
	}
	return "N/A"
}
