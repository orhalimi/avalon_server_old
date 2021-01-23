package main

import (
	"log"
	"math/rand"
	"time"
)

type SirPick struct {
	Options []string `json:"options,omitempty"`
	Pick    string   `json:"pick,omitempty"`
}


type SirMessageInternal struct {
	Pick string `json:"pick"`
}

type SirMessage struct {
	Tp      string             `json:"type"`
	Content SirMessageInternal `json:"content"`
}



func HandleSir(m SirMessageInternal) {
	globalMutex.Lock()
	pick := m.Pick
	character := globalBoard.PlayerToCharacter[PlayerName{pick}]
	SirPlayer := globalBoard.CharacterToPlayer[Seer]
	globalBoard.Secrets[SirPlayer.Player] = append(globalBoard.Secrets[SirPlayer.Player], pick+" is "+character)


	if BlanchefleurPlayer, ok := globalBoard.CharacterToPlayer[Blanchefleur]; ok  {
		seerMap := make(map[string]bool)
		seerMap[pick] = true
		globalBoard.whoSeeWho[Seer] = seerMap

		secrets := make([]string, 0)
		keys := make([]string, 0)

		for k := range globalBoard.whoSeeWho {
			if globalBoard.whoSeeWho[k] != nil && len(globalBoard.whoSeeWho[k]) > 0 {
				keys = append(keys, k)
			}

		}
		log.Println(" WhoSeeWho keys with values    =     ", keys)


		var See string

		isFound:=false
		random1 := rand.Intn(len(keys))
		var TruePlayer PlayerName
		var TrueCharacter string
		for !isFound {
			log.Println(" random1    =     ", random1)
			TrueCharacter = keys[random1]
			TruePlayer = globalBoard.CharacterToPlayer[TrueCharacter]
			log.Println(" TrueCharacter    =     ", TrueCharacter)
			log.Println(" TruePlayer    =     ", TruePlayer)
			random2 := rand.Intn(len(globalBoard.whoSeeWho[keys[random1]]))
			log.Println(" random2    =     ", random2)
			i :=0
			for k := range globalBoard.whoSeeWho[keys[random1]] {
				if i == random2 {
					if k != BlanchefleurPlayer.Player {
						See = k
						isFound = true
						break
					} else {
						random2 ++
					}
				}
				i++
			}
			random1 = (random1 + 1) % len(keys)
		}

		secrets = append(secrets, TruePlayer.Player + " see " + See)
		log.Println(TruePlayer.Player + " see " + See)

		random3 := rand.Intn(len(globalBoard.Characters))
		log.Println("random3 = ", random3)
		for globalBoard.Characters[random3] == TrueCharacter || globalBoard.Characters[random3] == Blanchefleur {
			random3 = (random3 + 1) % len(globalBoard.Characters)
		}
		log.Println("character for unseen = ", globalBoard.Characters[random3])
		unseenplayers := make([]string, 0)
		for _, p := range globalBoard.PlayerNames {
			if p == BlanchefleurPlayer || globalBoard.PlayerToCharacter[p] == globalBoard.Characters[random3] {
				continue
			}
			if globalBoard.whoSeeWho[globalBoard.Characters[random3]] == nil || len(globalBoard.whoSeeWho[globalBoard.Characters[random3]]) == 0 {
				unseenplayers = append(unseenplayers, p.Player)
				log.Println("found unseen = ", p.Player)
			} else {
				if _, ok := globalBoard.whoSeeWho[globalBoard.Characters[random3]][p.Player]; !ok  {
					unseenplayers = append(unseenplayers, p.Player)
					log.Println("found unseen = ", p.Player)
				}
			}
		}

		log.Println("unseens all = ", unseenplayers)
		FalseCharacter := globalBoard.Characters[random3]
		FalsePlayer := globalBoard.CharacterToPlayer[FalseCharacter]
		random4 := rand.Intn(len(unseenplayers))
		log.Println("random4 = ", random4)
		secrets = append(secrets, FalsePlayer.Player + " see " + unseenplayers[random4])

		rand.Seed(int64(time.Now().Nanosecond()))
		rand.Shuffle(len(secrets), func(i, j int) {
			secrets[i], secrets[j] = secrets[j], secrets[i]
		})
		log.Println("secrets = ", secrets)
		globalBoard.Secrets[BlanchefleurPlayer.Player] = secrets
	}

	globalBoard.State = WaitingForSuggestion
	globalMutex.Unlock()
}
