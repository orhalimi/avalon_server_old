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
	current := globalBoard.quests.current
	mp := globalBoard.quests.playersVotes[current]
	res := globalBoard.quests.results[current+1]
	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	if len(excaliburPick) == 1 {
		character := globalBoard.PlayerToCharacter[PlayerName{excaliburPick[0]}]
		curEntry.ExcaliburChosenPlayer = excaliburPick[0]
		playerVote := globalBoard.quests.playerVotedForCurrent[excaliburPick[0]]
		globalBoard.suggestions.excalibur.ChosenPlayerVote = playerVote
		globalBoard.Secrets[globalBoard.suggestions.excalibur.Player] = append(globalBoard.Secrets[globalBoard.suggestions.excalibur.Player], excaliburPick[0]+" voted "+getVoteStr(playerVote)+"(Quest "+strconv.FormatFloat(float64(curEntry.Id), 'f', 2, 32)+")")
		if character != Maeve {
			var newVote int
			log.Println("character:", character, "player vote:", playerVote)
			if playerVote == 2 {
				res.NumOfReversal--
				curEntry.NumberOfReversal--
				if character == GoodAngel {
					res.NumOfFailures++
					curEntry.NumberOfFailures++
					log.Println("new vote fail")
					newVote = 0 /*Fail*/
				} else if character == BadAngel {
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
	}
	EndJourney(&res, mp, &curEntry, current)
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry
	globalBoard.quests.results[current+1] = res
	globalBoard.quests.playersVotes[current] = mp
	globalBoard.quests.current++
	globalMutex.Unlock()
}
