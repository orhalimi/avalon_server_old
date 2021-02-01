package main

import "log"

type LadySuggestMessage struct {
	Tp      string `json:"type"`
	Content string `json:"content"`
}

type LadyResponseMessage struct {
	Tp      string `json:"type"`
	Content int    `json:"content"`
}

type LadyPublishResponseMessage struct {
	Tp      string `json:"type"`
	Content int    `json:"content"`
}

type LadyStats struct {
	currentSuggester    string
	currentChosenPlayer string
	previousSuggester string
	ladyResponse        int
}

func getOptionalLoyalty(player string) []string {
	character := globalBoard.PlayerToCharacter[PlayerName{player}]
	if character == QueenMab {
		return []string{"Bad", "Good"}
	}

	if character == Lot {
		return []string{"Bad"}
	}

	if character == Meliagant {
		return []string{"Bad"}
	}

	if character == "Gawain" || character == "Ginerva" {
		return []string{"Bad"}
	}

	if character == "Puck" {
		return []string{"Good"}
	}

	if character == Raven {
		return []string{"Bad"}
	}

	if _, ok := badCharacters[character]; ok {
		return []string{"Bad"}
	}

	if _, ok := goodCharacters[character]; ok {
		return []string{"Good"}
	}

	return []string{"Neutral"}
}

func LadySuggestHandler(suggestion string) {
	log.Println("got lady suggestion:", suggestion)
	globalMutex.Lock()
	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	curEntry.LadyChosenPlayer = suggestion
	curEntry.LadySuggester = globalBoard.ladyOfTheLake.currentSuggester
	globalBoard.archive[len(globalBoard.archive)-1] = curEntry
	globalBoard.ladyOfTheLake.currentChosenPlayer = suggestion

	globalBoard.State = LadyResponse
	globalBoard.StateDescription = "Lady Of The Lake: " + suggestion + " got The Lady. Waiting for his answer..."
	globalMutex.Unlock()
}

func LadyResponseHandler(loyalty int) {
	log.Println("got lady response:", loyalty)
	globalMutex.Lock()
	globalBoard.State = LadySuggesterPublishResponseToWorld
	globalBoard.ladyOfTheLake.ladyResponse = loyalty
	globalBoard.StateDescription = "Lady Of The Lake: " + globalBoard.ladyOfTheLake.currentSuggester + " got response from " + globalBoard.ladyOfTheLake.currentChosenPlayer +". Waiting for his publication..."
	globalMutex.Unlock()
}

func LadyPublishResponseHandler(loyalty int) {
	log.Println("got lady publish response:", loyalty)
	globalMutex.Lock()
	curEntry := globalBoard.archive[len(globalBoard.archive)-1] //Stats table
	if loyalty == 1 {
		curEntry.LadySuggesterPublishToTheWorld = "Good"
	} else {
		curEntry.LadySuggesterPublishToTheWorld = "Bad"
	}

	globalBoard.archive[len(globalBoard.archive)-1] = curEntry

	globalBoard.State = WaitingForSuggestion
	globalBoard.ladyOfTheLake.previousSuggester = globalBoard.ladyOfTheLake.currentSuggester
	globalBoard.ladyOfTheLake.currentSuggester = globalBoard.ladyOfTheLake.currentChosenPlayer
	globalBoard.ladyOfTheLake.currentChosenPlayer = ""
	globalBoard.ladyOfTheLake.ladyResponse = -1
	globalMutex.Unlock()
}
