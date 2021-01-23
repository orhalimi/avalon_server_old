package main

import "log"


type PlayerNameMurder struct {
	Player          string `json:"player,omitempty"`
	Ch              bool   `json:"ch,omitempty"`
	CharacterToKill string `json:"characterToKill,omitempty"`
}


type MurderMessage struct {
	Tp      string                `json:"type"`
	Content MurderMessageInternal `json:"content"`
}

type MurderMessageInternal struct {
	CharacterKill string             `json:"assassinkill"`
	Rest          []PlayerNameMurder `json:"rest"`
}

type MurderResult struct {
	target          []string
	targetCharacter []string
	by              string
	byCharacter     string
	success         bool
}

type MurderInfo struct {
	by []string
}

type Murder struct {
	target            []string
	TargetCharacters  []string `json:"target"`
	By                string   `json:"by"`
	ByCharacter       string   `json:"byCharacter"`
	stopIfSucceeded   bool
	StateAfterSuccess int
}

func HandleMurder(m MurderMessageInternal) {
	var curMurder Murder
	selection := m.Rest
	characterToKill := m.CharacterKill
	log.Println("selection: ", selection)
	if globalBoard.State == MurdersAfterGoodVictory {
		curMurder = globalBoard.PendingMurders[0]
	} else if globalBoard.State == MurdersAfterBadVictory {
		curMurder = globalBoard.PendingMurders[0]
	}

	chosenPlayers := make([]string, 0)
	for _, player := range selection {
		if player.Ch {
			if globalBoard.PlayerToCharacter[PlayerName{player.Player}] == SirGawain {
				globalBoard.State = VictoryForSirGawain
				return
			}
			chosenPlayers = append(chosenPlayers, player.Player)
			murderInfo, ok := globalBoard.PlayerToMurderInfo[player.Player]
			if ok {
				murderInfo.by = append(murderInfo.by, curMurder.By)
				globalBoard.PlayerToMurderInfo[player.Player] = murderInfo
			} else {
				globalBoard.PlayerToMurderInfo[player.Player] = MurderInfo{by: []string{curMurder.By}}
			}
		}
	}

	globalBoard.PendingMurders = globalBoard.PendingMurders[1:]
	murderResult := MurderResult{targetCharacter: curMurder.TargetCharacters, byCharacter: curMurder.ByCharacter}

	var isSuccess bool
	if curMurder.ByCharacter == Assassin {
		if len(chosenPlayers) == 1 {
			if characterToKill == globalBoard.PlayerToCharacter[PlayerName{chosenPlayers[0]}] {
				isSuccess = true
				log.Println("assassin murder success. chosenPlayers ", chosenPlayers[0])
			} else {
				log.Println("assassin murder failed. chosen Player is ", chosenPlayers[0], " with role ", globalBoard.PlayerToCharacter[PlayerName{chosenPlayers[0]}], "instead of ", characterToKill)
			}
		}
		if len(chosenPlayers) == 2 && characterToKill == "The-Lovers"{
			tristan, _ := globalBoard.CharacterToPlayer[Tristan]
			iseult, _ := globalBoard.CharacterToPlayer[Iseult]
			theLoversSlice := []string{tristan.Player, iseult.Player}
			if sameStringSlice(chosenPlayers, theLoversSlice) {
				isSuccess = true
				log.Println("assassin murdered the lovers successfully. players: ", chosenPlayers)
			}
		}
	} else {
		isSuccess = sameStringSlice(curMurder.target, chosenPlayers)
	}

	if isSuccess {
		//murder succeeded!
		log.Println("Murder Success! Killer:", curMurder.By, " Selection: ", selection)
		murderResult.success = true
		murderResult.byCharacter = curMurder.ByCharacter
		murderResult.target = chosenPlayers
		if curMurder.StateAfterSuccess != 0 {
			oldState := globalBoard.State
			globalBoard.State = curMurder.StateAfterSuccess
			log.Println("New State:", globalBoard.State)
			if oldState == MurdersAfterBadVictory && globalBoard.State == MurdersAfterGoodVictory {
				pendingMurders, hasMurders := GetMurdersAfterGoodsWins()
				if !hasMurders {
					globalBoard.State = VictoryForGood
					globalBoard.PendingMurders = make([]Murder, 0)
					return
				} else {
					globalBoard.PendingMurders = pendingMurders
				}
			}
		}
	} else {
		murderResult.success = false
	}

	if len(globalBoard.PendingMurders) == 0 {
		log.Println("No more murders")
		if globalBoard.State == MurdersAfterGoodVictory {
			globalBoard.State = VictoryForGood
		} else if globalBoard.State == MurdersAfterBadVictory {
			globalBoard.State = VictoryForBad
		}
	}
}

func GetMurdersAfterGoodsWins() ([]Murder, bool) {

	murders := make([]Murder, 0)

	if beast, isKingClaudinExists := globalBoard.CharacterToPlayer[TheQuestingBeast]; isKingClaudinExists {
		if pellinore, isPrinceClaudinExists := globalBoard.CharacterToPlayer[Pellinore]; isPrinceClaudinExists {
			m := Murder{target: []string{beast.Player}, TargetCharacters: []string{TheQuestingBeast}, By: pellinore.Player}
			murders = append(murders, m)
		}
	}

	if _, isKingClaudinExists := globalBoard.CharacterToPlayer[KingClaudin]; isKingClaudinExists {
		if _, isPrinceClaudinExists := globalBoard.CharacterToPlayer[PrinceClaudin]; isPrinceClaudinExists {
			if percivalPlayerName, isPercivalExists := globalBoard.CharacterToPlayer[Percival]; isPercivalExists {
				m := Murder{target: getAllBads(), TargetCharacters: getAllBadsChars(), By: percivalPlayerName.Player, StateAfterSuccess: VictoryForGood}
				murders = append(murders, m)
			} else if arthurPlayerName, isArthurExists := globalBoard.CharacterToPlayer[KingArthur]; isArthurExists {
				m := Murder{target: getAllBads(), TargetCharacters: getAllBadsChars(), By: arthurPlayerName.Player, StateAfterSuccess: VictoryForGood}
				murders = append(murders, m)
			}
		}
	}

	merlinAppenticePlayerName, ok := globalBoard.CharacterToPlayer[MerlinApprentice]
	targetCharacters := make([]string, 0)
	targetSlice := make([]string, 0)
	if ok {
		targetCharacters = append(targetCharacters, MerlinApprentice)
		targetSlice = append(targetSlice, merlinAppenticePlayerName.Player)
	}
	assassin := globalBoard.CharacterToPlayer[Assassin]

	if merlinPlayerName, isMerlinExists := globalBoard.CharacterToPlayer[Merlin]; isMerlinExists {
		targetSlice = append(targetSlice, merlinPlayerName.Player)
		targetCharacters = append(targetCharacters, Merlin)
	}
	if vivianPlayerName, isVivianExists := globalBoard.CharacterToPlayer[Viviana]; isVivianExists {
		targetSlice = append(targetSlice, vivianPlayerName.Player)
		targetCharacters = append(targetCharacters, Viviana)
	}
	if nirlemPlayerName, isNirlemExists := globalBoard.CharacterToPlayer[Nirlem]; isNirlemExists {
		targetSlice = append(targetSlice, nirlemPlayerName.Player)
		targetCharacters = append(targetCharacters, Nirlem)
	}

	if tristan, isTristanExists := globalBoard.CharacterToPlayer[Tristan]; isTristanExists {
		if iseult, isIseultExists := globalBoard.CharacterToPlayer[Iseult]; isIseultExists {
			targetSlice = append(targetSlice, tristan.Player)
			targetSlice = append(targetSlice, iseult.Player)
			targetCharacters = append(targetCharacters, "The-Lovers")
		}
	}

	m := Murder{target: targetSlice, TargetCharacters: targetCharacters, By: assassin.Player, ByCharacter: Assassin, StateAfterSuccess: VictoryForBad}
	murders = append(murders, m)

	return murders, len(murders) > 0
}

func GetMurdersAfterBadsWins() ([]Murder, bool) {

	murders := make([]Murder, 0)

	if beast, isKingClaudinExists := globalBoard.CharacterToPlayer[TheQuestingBeast]; isKingClaudinExists {
		if pellinore, isPrinceClaudinExists := globalBoard.CharacterToPlayer[Pellinore]; isPrinceClaudinExists {
			m := Murder{target: []string{beast.Player}, TargetCharacters: []string{TheQuestingBeast}, By: pellinore.Player}
			murders = append(murders, m)
		}
	}

	if cordana, isKingClaudinExists := globalBoard.CharacterToPlayer[Cordana]; isKingClaudinExists {
		if mordred, isPrinceClaudinExists := globalBoard.CharacterToPlayer[Mordred]; isPrinceClaudinExists {
			m := Murder{target: []string{mordred.Player}, TargetCharacters: []string{Cordana}, By: cordana.Player, StateAfterSuccess: MurdersAfterGoodVictory}
			murders = append(murders, m)
		}
	}

	if kingArthur, isKingArthurExists := globalBoard.CharacterToPlayer[KingArthur]; isKingArthurExists {
		m := Murder{target: getAllBads(), TargetCharacters: getAllBadsChars(), By: kingArthur.Player, StateAfterSuccess: VictoryForGood}
		murders = append(murders, m)
	}


	return murders, len(murders) > 0
}

func getAllBadsChars() []string {
	allBads := make([]string, 0)
	for _, player := range globalBoard.PlayerNames {
		if ch, ok := globalBoard.PlayerToCharacter[player]; ok {
			if _, ok := badCharacters[ch]; ok {
				allBads = append(allBads, ch)
			}
		}
	}
	return allBads
}

func getAllBads() []string {
	allBads := make([]string, 0)
	for _, player := range globalBoard.PlayerNames {
		if ch, ok := globalBoard.PlayerToCharacter[player]; ok {
			if _, ok := badCharacters[ch]; ok {
				allBads = append(allBads, player.Player)
			}
		}
	}
	return allBads
}