package main

import (
	"log"
	"math/rand"
	"time"
)



type StartGameMessage struct {
	Tp      string            `json:"type"`
	Content GameConfiguration `json:"content"`
}

type Ch struct {
	Name     string `json:"name"`
	Checked  bool   `json:"checked"`
	Assassin bool   `json:"assassin"`
}

type GameConfiguration struct {
	Characters []Ch `json:"characters"`
	Excalibur  bool `json:"excalibur"`
	Lady       bool `json:"lady"`
}

func StartGameHandler(newGameConfig GameConfiguration) {
	log.Println("newGameConfig", newGameConfig)
	globalMutex.Lock()

	chosenCharacters := make([]string, 0)
	numOfPlayers := len(globalBoard.PlayerNames)
	requiredBads := globalConfigPerNumOfPlayers[numOfPlayers].NumOfBadCharacters

	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(globalBoard.PlayerNames), func(i, j int) {
		globalBoard.PlayerNames[i], globalBoard.PlayerNames[j] = globalBoard.PlayerNames[j], globalBoard.PlayerNames[i]
	})

	if newGameConfig.Excalibur == true {
		globalBoard.quests.Flags[EXCALIBUR] = true
		log.Println("excalibur - on ")
	}

	if newGameConfig.Lady == true {
		globalBoard.quests.Flags[LADY] = true
		globalBoard.ladyOfTheLake.currentSuggester = globalBoard.PlayerNames[len(globalBoard.PlayerNames)-1].Player
		log.Println("lady - on ")
	}

	globalBoard.lancelotCards = []int{0, 0, 1, 0, 1, 0, 0}
	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(globalBoard.lancelotCards), func(i, j int) {
		globalBoard.lancelotCards[i], globalBoard.lancelotCards[j] = globalBoard.lancelotCards[j], globalBoard.lancelotCards[i]
	})
	log.Println("===========", globalBoard.lancelotCards)
	var numOfBads int
	var numOfGood int
	var hasEctor bool
	for _, v := range newGameConfig.Characters {
		if v.Checked == true {
			if v.Name == Ector {
				hasEctor = true //need to use smaller board game in this case
			}
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
	log.Println("chosen characters: ", chosenCharacters)
	if _, ok := globalBoard.CharacterToPlayer[Seer]; ok {
		globalBoard.State = SirPickPlayer
	} else {
		globalBoard.State = WaitingForSuggestion
	}
	if globalBoard.quests.results == nil {
		globalBoard.quests.results = make(map[int]QuestStats)
	}

	//Ector
	if hasEctor {
		globalBoard.numOfPlayers = len(globalBoard.PlayerNames) - 1
	} else {
		globalBoard.numOfPlayers = len(globalBoard.PlayerNames)
	}
	globalBoard.Characters = chosenCharacters
	var hasMeliagant bool
	if _, ok := globalBoard.CharacterToPlayer[Meliagant]; ok {
		hasMeliagant = true
	}
	for i := 0; i < globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].NumOfQuests; i++ {
		en := QuestStats{}
		log.Println(i)
		en.Ppp = getTypeOfLevel(i+1, len(globalBoard.PlayerNames))
		en.NumOfPlayers = globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].PlayersPerLevel[i]
		if hasMeliagant {
			en.NumOfPlayers--
		}
		globalBoard.quests.results[i+1] = en
		log.Println(en)
	}
	globalBoard.suggestions.suggesterIndex = 0

	numOfUnsuccesfulRetries := globalConfigPerNumOfPlayers[globalBoard.numOfPlayers].RetriesPerLevel[globalBoard.quests.current]
	suggesterVetoIn := (globalBoard.suggestions.suggesterIndex + numOfUnsuccesfulRetries - 1) % len(globalBoard.PlayerNames)
	globalBoard.suggestions.PlayerWithVeto = globalBoard.PlayerNames[suggesterVetoIn].Player

	WhoSeeWho := make(map[string]map[string]bool)
	for _, player := range globalBoard.PlayerNames {
		globalBoard.Secrets[player.Player], globalBoard.whoSeeWho = GetSecretsFromPlayerName(player, WhoSeeWho)
		log.Println(player, " Secrets     =     ", globalBoard.Secrets[player.Player])
		log.Println(player, " WhoSeeWho     =     ", WhoSeeWho)
	}

	_, hasSeer := globalBoard.CharacterToPlayer[Seer]
	if BlanchefleurPlayer, ok := globalBoard.CharacterToPlayer[Blanchefleur]; ok && !hasSeer {
		secrets := make([]string, 0)
		keys := make([]string, 0)

		for k := range WhoSeeWho {
			if WhoSeeWho[k] != nil && len(WhoSeeWho[k]) > 0 {
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
			random2 := rand.Intn(len(WhoSeeWho[keys[random1]]))
			log.Println(" random2    =     ", random2)
			i :=0
			for k := range WhoSeeWho[keys[random1]] {
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
			if WhoSeeWho[globalBoard.Characters[random3]] == nil || len(WhoSeeWho[globalBoard.Characters[random3]]) == 0 {
				unseenplayers = append(unseenplayers, p.Player)
				log.Println("found unseen = ", p.Player)
			} else {
				if _, ok := WhoSeeWho[globalBoard.Characters[random3]][p.Player]; !ok  {
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

	_, hasBadLancelot := globalBoard.CharacterToPlayer[LancelotBad]
	_, hasGoodLancelot := globalBoard.CharacterToPlayer[LancelotGood]
	if hasBadLancelot && hasGoodLancelot {
		globalBoard.quests.Flags[HAS_TWO_LANCELOT] = true
	} else if hasBadLancelot {
		globalBoard.quests.Flags[HAS_ONLY_BAD_LANCELOT] = true
	} else if hasGoodLancelot {
		globalBoard.quests.Flags[HAS_ONLY_GOOD_LANCELOT] = true
	}

	_, hasBalain := globalBoard.CharacterToPlayer[Balain]
	_, hasBalin := globalBoard.CharacterToPlayer[Balin]
	if hasBalain && hasBalin {
		globalBoard.quests.Flags[HAS_BALAIN_AND_BALIN] = true
	}


	globalBoard.CharacterToPlayer[Assassin] = PlayerName{assassinPlayer}
	globalMutex.Unlock()
}


func GetSecretsFromPlayerName(player PlayerName, whoSeeWho map[string]map[string]bool) ([]string, map[string]map[string]bool) {

	secrets := make([]string, 0)
	if player.Player == "" {
		return nil, whoSeeWho
	}

	strayPlayer, _ := globalBoard.CharacterToPlayer[Stray]
	character := globalBoard.PlayerToCharacter[player]


	if character == Gornemant {
		bads := make([]string, 0)
		goods := make([]string, 0)
		for _, c := range globalBoard.Characters {
			if _, ok := goodCharacters[c]; ok {
				goods = append(goods, c)
			} else {
				bads = append(bads, c)
			}
		}
		sameTeamTakenFromBads := rand.Intn(2)
		var sameTeam []string
		var notSameTeam []string
		if sameTeamTakenFromBads == 1 {
			sameTeam = bads
			notSameTeam = goods
			log.Println("sameTeamTakenFromBads == 1")
		} else {
			sameTeam = goods
			notSameTeam = bads
		}
		log.Println("sameTeam =", sameTeam, "len=", len(sameTeam))
		log.Println("notSameTeam =", notSameTeam, "len=", len(notSameTeam))

		//remove this player from both arrays!!!!
		idx := 0
		isFound:=false
		for _, s := range sameTeam {
			if s != player.Player {
				idx++
			} else {
				isFound = true
				break
			}
		}
		if isFound {
			sameTeam = removeElementFromStringSlice(sameTeam, idx)
		}

		idx = 0
		isFound= false
		for _, s := range notSameTeam {
			if s != player.Player {
				idx++
			} else {
				isFound = true
				break
			}
		}
		if isFound {
			notSameTeam = removeElementFromStringSlice(notSameTeam, idx)
			//end of this removal
		}
		random1 := rand.Intn(len(sameTeam))
		random2 := rand.Intn(len(sameTeam))
		Player1 := globalBoard.CharacterToPlayer[sameTeam[random1]].Player
		Player2 := globalBoard.CharacterToPlayer[sameTeam[random2]].Player
		if random2 == random1 {
			random2 = (random2 + 1) % len(sameTeam)
			Player2 = globalBoard.CharacterToPlayer[sameTeam[random2]].Player
		}

		log.Println("random1 =", random1, " random2 = " , random2)

		secrets = append(secrets, Player1 + " and " + Player2)
		sameTeam = removeElementFromStringSlice(sameTeam, random1)

		idx = 0
		isFound = false
		for i, _ := range sameTeam {
			if Player2 != globalBoard.CharacterToPlayer[sameTeam[i]].Player {
				idx++
			} else {
				isFound = true
				break
			}
		}
		if isFound {
			sameTeam = removeElementFromStringSlice(sameTeam, idx)
		}

		log.Println("new sameTeam =", sameTeam)
		random3 := rand.Intn(len(notSameTeam))
		random4 := rand.Intn(len(sameTeam))
		log.Println("random3 =", random3, " random4 = " , random4)
		Player3 := globalBoard.CharacterToPlayer[notSameTeam[random3]].Player
		Player4 := globalBoard.CharacterToPlayer[sameTeam[random4]].Player
		secrets = append(secrets, Player3 + " and " + Player4)
	}

	if character == Meliagant {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}

		for k, v := range globalBoard.CharacterToPlayer {

			if _, ok := badCharacters[k]; ok {
				if v.Player == strayPlayer.Player {
					secrets = append(secrets, v.Player+" is Stray")
				} else {
					secrets = append(secrets, v.Player+" is " + k)
				}

				mapp[v.Player] = true
			} else {
				if k == Lot {
					secrets = append(secrets, v.Player+" is Lot")
					mapp[v.Player] = true
				}
			}
		}
		whoSeeWho[Meliagant] = mapp
	}







	if character == Merlin {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}


		for k, v := range globalBoard.CharacterToPlayer {

			if _, ok := badCharacters[k]; ok && k != Mordred && k != Accolon {
				if k == Oberon {
					secrets = append(secrets, v.Player+" is Oberon")
					mapp[v.Player] = true
				} else {
					secrets = append(secrets, v.Player+" is bad")
					mapp[v.Player] = true
				}
			}
			if k == Stray {
				secrets = append(secrets, v.Player+" is Stray")
				mapp[v.Player] = true
			}
			if k == Lot {
				secrets = append(secrets, v.Player+" is Lot")
				mapp[v.Player] = true
			}
			if k == Meliagant {
				secrets = append(secrets, v.Player+" is bad")
				mapp[v.Player] = true
			}
			if k == "Ginerva" {
				secrets = append(secrets, v.Player+" is bad")
				mapp[v.Player] = true
			}
			if k == SirKay {
				secrets = append(secrets, v.Player+" is bad")
				mapp[v.Player] = true
			}
			if k == "Gawain" {
				secrets = append(secrets, v.Player+" is Gawain")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[Merlin] = mapp
	}
	if _, ok := goodCharacters[character]; ok && character != Nirlem && character != Lot && character != Meliagant {
		if nirlem, ok := globalBoard.CharacterToPlayer[Nirlem]; ok && character != LancelotGood && character != Balain {
			mapp := whoSeeWho[character]
			if mapp == nil {
				mapp= make (map[string]bool)
			}
			secrets = append(secrets, nirlem.Player+" is Nirlem")
			mapp[nirlem.Player] = true
			whoSeeWho[character] = mapp
		}
	}
	if character == Guinevere {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == LancelotGood {
				secrets = append(secrets, v.Player+" is Lancelot")
				mapp[v.Player] = true
			}
			if k == LancelotBad {
				secrets = append(secrets, v.Player+" is Lancelot")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Iseult {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Tristan {
				secrets = append(secrets, v.Player+" is Tristan")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Balin {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Balain {
				secrets = append(secrets, v.Player+" is Balain")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Balain {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Balin {
				secrets = append(secrets, v.Player+" is Balin")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == PrinceClaudin {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == KingClaudin {
				secrets = append(secrets, v.Player+" is King-Claudin")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == KingClaudin {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == PrinceClaudin {
				secrets = append(secrets, v.Player+" is Prince-Claudin")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}

	if character == MerlinApprentice {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Percival {
				secrets = append(secrets, v.Player+" is Percival/Assasin")
				mapp[v.Player] = true
			}
			if k == Assassin {
				secrets = append(secrets, v.Player+" is Percival/Assassin")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Tristan {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Iseult {
				secrets = append(secrets, v.Player+" is Iseult")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Lot {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; (ok && k != character && k != Oberon && k != Accolon) || k == Meliagant {
				if k == "Polygraph" {
					secrets = append(secrets, v.Player+" is polygraph")
					mapp[v.Player] = true
				} else {
					secrets = append(secrets, v.Player+" is bad")
					mapp[v.Player] = true
				}
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Nimue {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Galahad {
				secrets = append(secrets, v.Player+" is Galahad")
				mapp[v.Player] = true
			}
			if k == Merlin {
				secrets = append(secrets, v.Player+" is Merlin")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Nerzhul {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Oberon {
				secrets = append(secrets, v.Player + " is Oberon")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Dagonet {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Oberon {
				secrets = append(secrets, v.Player + " is Oberon")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Morgana {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == "Gawain" {
				secrets = append(secrets, v.Player+" is Gawain")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}
	if character == Percival {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Morgana {
				if _, ok := globalBoard.CharacterToPlayer[Merlin]; !ok {
					secrets = append(secrets, v.Player+" is Morgana/Viviana")
					mapp[v.Player] = true
				} else {
					secrets = append(secrets, v.Player+" is Morgana/Merlin")
					mapp[v.Player] = true
				}
			}
			if k == Merlin {
				secrets = append(secrets, v.Player+" is Morgana/Merlin")
				mapp[v.Player] = true
			}
			if k == Viviana {
				if _, ok := globalBoard.CharacterToPlayer[Merlin]; !ok {
					secrets = append(secrets, v.Player+" is Morgana/Viviana")
					mapp[v.Player] = true
				}
			}

		}
		whoSeeWho[character] = mapp
	}
	if _, ok := badCharacters[character]; ok && character != Oberon && character != Accolon && character != LancelotBad && character != Balin && character != Agravain {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; (ok && k != character && k != Oberon && k != Accolon && k != Agravain) || k == Meliagant  {
				if k == "Polygraph" {
					secrets = append(secrets, v.Player+" is polygraph")
					mapp[v.Player] = true
				} else {

					if v.Player == strayPlayer.Player  {
						secrets = append(secrets, v.Player+" is Stray")
					} else {
						secrets = append(secrets, v.Player+" is bad")
					}
					mapp[v.Player] = true
				}
			}
		}
		if  _, ok := globalBoard.CharacterToPlayer[Stray]; ok {
			if !mapp[strayPlayer.Player] {
				secrets = append(secrets, strayPlayer.Player+" is Stray")
				mapp[strayPlayer.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}

	if character == TheQuestingBeast {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if k == Pellinore {
				secrets = append(secrets, v.Player+" is Pellinore")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}

	if character == "Gawain" {
		mapp := whoSeeWho[character]
		if mapp == nil {
			mapp= make (map[string]bool)
		}
		for k, v := range globalBoard.CharacterToPlayer {
			if _, ok := badCharacters[k]; (ok && k != character && k != Oberon && k != Accolon) || k == Meliagant {
				secrets = append(secrets, v.Player+" ")
				mapp[v.Player] = true
			}
			if k == Percival || k == Merlin || k == Nirlem || k == Viviana {
				secrets = append(secrets, v.Player+" ")
				mapp[v.Player] = true
			}
		}
		whoSeeWho[character] = mapp
	}

	rand.Seed(int64(time.Now().Nanosecond()))
	rand.Shuffle(len(secrets), func(i, j int) {
		secrets[i], secrets[j] = secrets[j], secrets[i]
	})
	return secrets, whoSeeWho
}


func assignCharactersToRegisteredPlayers(newGameConfig []Ch, chosenCharacters []string) ([]string, string) {
	var assassinCharacter string
	var hasStray bool
	for _, v := range newGameConfig {
		if v.Checked == true {
			if v.Name == Stray {
				hasStray = true
			}
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


	if hasStray {
		strayPlayer := globalBoard.CharacterToPlayer[Stray]
		goodChars := make([]string, 0)
		for _, c := range optionalGoodsForStray {
			if _, ok := globalBoard.CharacterToPlayer[c]; !ok {
				if len(globalBoard.PlayerNames) >= 7 || !notAllowedGoodsForStrayForLessThan7Players[c] {
					goodChars = append(goodChars, c)
				}
			}
		}
		rand.Seed(int64(time.Now().Nanosecond()))
		rand.Shuffle(len(goodChars), func(i, j int) {
			goodChars[i], goodChars[j] = goodChars[j], goodChars[i]
		})
		random1 := rand.Intn(len(goodChars))

		newCharactersForStray := []string{Mordred, goodChars[random1]}
		random2 := rand.Intn(len(newCharactersForStray))
		globalBoard.PlayerToCharacter[strayPlayer] = newCharactersForStray[random2]
		globalBoard.CharacterToPlayer[newCharactersForStray[random2]] = strayPlayer
		//globalBoard.Characters = append(globalBoard.Characters, newCharactersForStray[random2])
		//so we have character['stray'] --> playerX and playerX --> NEW CHARACTER
	}
	if _, ok := globalBoard.CharacterToPlayer[Assassin]; ok {
		assassinCharacter = Assassin
	}
	return chosenCharacters, globalBoard.CharacterToPlayer[assassinCharacter].Player
}
