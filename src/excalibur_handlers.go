package main

import (
	"log"
	"strconv"
)

type Excalibur struct {
	Player           string `json:"excalibur_player,omitempty"`
	Suggester        string `json:"suggester,omitempty"`
	ChosenPlayerVote int    `json:"vote,omitempty"`
}

type ExcaliburMessage struct {
	Tp      string   `json:"type"`
	Content []string `json:"content"`
}

func ExcaliburHandler(excaliburPick []string) {
	log.Println("got new excalibur pick:", excaliburPick)
	globalMutex.Lock()
	defer globalMutex.Unlock()

	if globalBoard.State != ExcaliburPick {
		return
	}

	current := globalBoard.quests.current
	mp := globalBoard.quests.playersVotes[current]
	res := globalBoard.quests.results[current+1]
	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table

	globalBoard.StateDescription = "Excalibur: " + globalBoard.suggestions.excalibur.Player +
		" chosed player to change his vote..."

	if len(excaliburPick) == 1 {
		character := globalBoard.PlayerToCharacter[PlayerName{excaliburPick[0]}]
		curEntry.ExcaliburChosenPlayer = excaliburPick[0]
		playerVote := globalBoard.quests.playerVotedForCurrent[excaliburPick[0]]
		globalBoard.suggestions.excalibur.ChosenPlayerVote = playerVote
		globalBoard.Secrets[globalBoard.suggestions.excalibur.Player] = append(globalBoard.Secrets[globalBoard.suggestions.excalibur.Player], excaliburPick[0]+" voted "+getVoteStr(playerVote)+"(Quest "+strconv.FormatFloat(float64(curEntry.Id), 'f', 2, 32)+")")

		if character != Maeve {
			var newVote int
			log.Println("character:", character, "player vote:", playerVote)
			if playerVote == VoteReversal {
				res.NumOfReversal--
				curEntry.NumberOfReversal--
				if character == GoodAngel {
					res.NumOfFailures++
					curEntry.NumberOfFailures++
					log.Println("new vote fail")
					newVote = VoteFail
				} else if character == BadAngel {
					res.NumOfSuccess++
					curEntry.NumberOfSuccesses++
					log.Println("new vote success")
					newVote = VoteSuccess
				}
			} else if playerVote == VoteFail || playerVote == VoteBeast {
				if playerVote == VoteFail {
					res.NumOfFailures--
					curEntry.NumberOfFailures--
				} else {
					res.NumOfBeasts--
					curEntry.NumberOfBeasts--
				}
				newVote = VoteSuccess
				log.Println("new vote success")
				res.NumOfSuccess++
				curEntry.NumberOfSuccesses++
			} else if playerVote == VoteSuccess {
				res.NumOfSuccess--
				curEntry.NumberOfSuccesses--
				curEntry.NumberOfFailures++
				res.NumOfFailures++
				newVote = VoteFail
				log.Println("new vote fail")
			} else if playerVote == VoteEmpty {
				res.NumOfEmpty--
				curEntry.NumberOfEmpty--
				curEntry.NumberOfSuccesses++
				res.NumOfSuccess++
				newVote = VoteSuccess
				log.Println("new vote success (original vote is empty)")
			}
			for i, vote := range mp {
				if vote == playerVote {
					mp[i] = newVote
					break
				}
			}
			globalBoard.quests.playerVotedForCurrent[excaliburPick[0]] = newVote
		}

		/* If we have Avalon Power, cancel this quest. */
		if StartNewSuggestion(mp, curEntry, current) {
			return
		}
	}
	EndJourney(&res, mp, &curEntry, current)
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry
	globalBoard.quests.results[current+1] = res
	globalBoard.quests.playersVotes[current] = mp
	globalBoard.quests.current++
}
