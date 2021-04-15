package main


const ( //game state
	NotStarted                          = iota
	SirPickPlayer                       = 1
	WaitingForSuggestion                = 2
	SuggestionVoting                    = 3
	JorneyVoting                        = 4
	ExcaliburPick                       = 5
	VictoryForGood                      = 6
	VictoryForBad                       = 7
	MurdersAfterGoodVictory             = 8
	MurdersAfterBadVictory              = 9
	VictoryForGawain                    = 10
	WaitingForLadySuggester             = 11
	LadyResponse                        = 12
	LadySuggesterPublishResponseToWorld = 13
	VictoryForSirGawain = 14
)

const (
	JorneyFail    = 1
	JorneySuccess = 2
)

const (
	RegularQuest          = 1
	FlushQuest            = 2
	TwoFailsRequiredQuest = 3
)

type boardConfigurations struct {
	NumOfQuests        int
	NumOfBadCharacters int
	PlayersPerLevel    []int
	RetriesPerLevel    []int
}

var globalConfigPerNumOfPlayers = map[int]boardConfigurations{
	1:  {3, 0, []int{1}, []int{5}},
	2:  {3, 1, []int{2, 2, 2}, []int{5, 5, 5}},
	4:  {4, 1, []int{2, 3, 3, 3}, []int{5, 5, 5, 5}},
	5:  {5, 2, []int{2, 3, 2, 3, 3}, []int{5, 5, 5, 5, 5}},
	6:  {5, 2, []int{2, 3, 4, 3, 4}, []int{5, 5, 5, 5, 5}},
	7:  {7, 3, []int{3, 3, 3, 4, 3, 4, 4}, []int{5, 5, 5, 7, 7, 7, 3}},
	8:  {7, 3, []int{3, 3, 4, 4, 4, 5, 5}, []int{5, 5, 5, 7, 7, 7, 3}},
	9:  {7, 3, []int{3, 4, 4, 5, 4, 5, 5}, []int{5, 5, 5, 7, 7, 7, 3}},
	10: {7, 4, []int{3, 4, 4, 5, 4, 5, 5}, []int{5, 5, 5, 7, 7, 7, 3}},
	11: {7, 4, []int{4, 5, 4, 5, 5, 5, 6}, []int{5, 5, 5, 7, 7, 7, 3}},
	12: {7, 5, []int{4, 5, 5, 6, 5, 6, 6}, []int{5, 5, 5, 7, 7, 7, 3}},
	13: {8, 5, []int{4, 5, 5, 6, 5, 6, 6, 7}, []int{5, 5, 5, 7, 7, 7, 7, 3}},
}

var neutralCharacters = map[string]bool{
	"Ginerva": true,
	"Puck":    true,
	"Gawain":  true,
	TheQuestingBeast: true,
}

var optionalGoodsForStray = []string{GoodAngel, Titanya, Nimue, Raven, KingArthur, SirRobin,
	TheCoward, MerlinApprentice, Guinevere, Gornemant, Blanchefleur, SirGawain, Elaine, LoyalServentOfArthur, LoyalServentOfArthurA, LoyalServentOfArthurB, LoyalServentOfArthurC, LoyalServentOfArthurD}

var notAllowedGoodsForStrayForLessThan7Players = map[string]bool{
	Gornemant:           true,
	Blanchefleur:       true,
}


const ( // Good Characters...
	Viviana = "Viviana"
	KingArthur = "King-Arthur"
	Seer = "Seer"
	Titanya = "Titanya"
	Galahad = "Galahad"
	Nimue = "Nimue"
	SirKay = "Sir-Kay"
	GoodAngel = "Good-Angel"
	Percival = "Percival"
	Merlin = "Merlin"
	Tristan = "Tristan"
	Iseult = "Iseult"
	PrinceClaudin = "Prince-Claudin"
	Nirlem = "Nirlem"
	SirRobin = "Sir-Robin"
	Pellinore = "Pellinore"
	Lot = "Lot"
	Cordana = "Cordana"
	TheCoward = "The-Coward"
	MerlinApprentice = "Merlin-Apprentice"
	LancelotGood = "Lancelot-Good"
	Guinevere = "Guinevere"
	Galaad = "Galaad"
	Raven = "Raven"
	Balain = "Balain"
	SirGawain = "Sir-Gawain"
	Jarvan = "Jarvan"
	Stray = "Stray"
	Ector = "Ector"
	Elaine = "Elaine"
	Blanchefleur = "Blanchefleur"
	TomThumb = "Tom-Thumb"
	Gornemant = "Gornemant"
	Dagonet = "Dagonet"
	Meliagant = "Meliagant"
	Bors = "Bors"
	UtherPendragon = "Uther-Pendragon"
	LoyalServentOfArthur = "Loyal-Servent-Of-Arthur"
	LoyalServentOfArthurA = "Loyal-Servent-Of-Arthur1"
	LoyalServentOfArthurB = "Loyal-Servent-Of-Arthur2"
	LoyalServentOfArthurC = "Loyal-Servent-Of-Arthur3"
	LoyalServentOfArthurD = "Loyal-Servent-Of-Arthur4"
)

const (
	Gawain = "Gawain"
	Puck = "Puck"
	Ginerva = "Ginerva"
)
const ( // Bad Characters...
	Morgana = "Morgana"
	Assassin = "Assassin"
	Mordred = "Mordred"
	Oberon = "Oberon"
	BadAngel = "Bad-Angel"
	KingClaudin = "King-Claudin"
	Polygraph = "Polygraph"
	TheQuestingBeast = "The-Questing-Beast"
	Accolon = "Accolon"
	LancelotBad = "Lancelot-Bad"
	QueenMab = "Queen-Mab"
	Balin = "Balin"
	Maeve = "Maeve"
	Agravain = "Agravain"
	Nerzhul = "Nerzhul"
	Mora = "Mora"
	Melwas = "Melwas"
	Claudas = "Claudas"
	MinionOfMordred = "Minion-Of-Mordred"
	MinionOfMordredA = "Minion-Of-Mordred1"
	MinionOfMordredB = "Minion-Of-Mordred2"
)

var goodCharacters = map[string]bool{
	Viviana:           true,
	KingArthur:       true,
	Seer:              true,
	Titanya:           true,
	Galahad:           true,
	Nimue:             true,
	SirKay:           true,
	GoodAngel:        true,
	Percival:          true,
	Merlin:            true,
	Tristan:           true,
	Iseult:            true,
	PrinceClaudin:    true,
	Nirlem:            true,
	SirRobin:         true,
	Pellinore:         true,
	Lot:               true,
	Cordana:           true,
	TheCoward:        true,
	MerlinApprentice: true,
	LancelotGood:     true,
	Guinevere:         true,
	Galaad:            true,
	Raven:             true,
	Balain:            true,
	SirGawain:        true,
	Jarvan:            true,
	Stray:             true,
	Ector:             true,
	Elaine:            true,
	Blanchefleur:      true,
	TomThumb:			true,
	Gornemant:         true,
	Dagonet:           true,
	Meliagant:         true,
	Bors:              true,
	UtherPendragon:		true,
	LoyalServentOfArthurA: true,
	LoyalServentOfArthurB: true,
	LoyalServentOfArthurC: true,
	LoyalServentOfArthurD: true,
	LoyalServentOfArthur: true,
}
var badCharacters = map[string]bool{
	Morgana:            true,
	Assassin:           true,
	Mordred:            true,
	Oberon:             true,
	BadAngel:          true,
	KingClaudin:       true,
	Polygraph:          true,
	Accolon:            true,
	LancelotBad:       true,
	QueenMab:          true,
	Balin:              true,
	Maeve:              true,
	Agravain:           true,
	Nerzhul:            true,
	Mora:               true,
	Melwas:             true,
	Claudas:			true,
	MinionOfMordred: 	true,
	MinionOfMordredA:	true,
	MinionOfMordredB:	true,
}

type PlayerName struct {
	Player string `json:"player,omitempty"`
}


type QuestStats struct {
	Final         int `json:"final,omitempty"`
	Ppp           int `json:"ppp,omitempty"`
	NumOfPlayers  int `json:"numofplayers,omitempty"`
	NumOfSuccess  int `json:"successes,omitempty"`
	NumOfReversal int `json:"reversals,omitempty"`
	NumOfFailures int `json:"failures,omitempty"`
	NumOfBeasts   int `json:"beasts,omitempty"`
	AvalonPower   bool `json:"avalon_power,omitempty"`
	NumOfEmpty int	`json:"empty,omitempty"`
}

const ( //Flags
	TITANYA_FIRST_FAIL = iota
	BEAST_FIRST_SUCCESS
	HAS_TWO_LANCELOT
	HAS_BALAIN_AND_BALIN
	HAS_ONLY_GOOD_LANCELOT
	HAS_ONLY_BAD_LANCELOT
	EXCALIBUR
	ELAINE_AVALON_POWER_CARD
	LADY
	BEAST_VOTE_SEEN
	BEAST_AND_PELLINORE_AT_SAME_QUEST
)

type QuestManager struct {
	current                    int //counts from 0
	playersVotes               [][]int
	Flags                      map[int]bool
	results                    map[int]QuestStats
	realResults                map[int]QuestStats
	successfulQuest            int
	unsuccessfulQuest          int
	playerVotedForCurrent      map[string]int
	playerVotedForCurrentQuest []string
	differentResults           map[int]int //for king arthur. mapping level to real result
}

type QuestArchiveItem struct {
	PlayersVotedYes                []string   `json:"playersAcceptedQuest"`
	PlayersVotedNo                 []string   `json:"playersNotAcceptedQuest"`
	Suggester                      PlayerName `json:"suggester"`
	SuggestedPlayers               []string   `json:"suggestedPlayers"`
	IsSuggestionAccepted           bool       `json:"isSuggestionAccepted"`
	IsSwitchLancelot               bool       `json:"switch"`
	NumberOfReversal               int        `json:"numberOfReversal"`
	NumberOfSuccesses              int        `json:"numberOfSuccesses"`
	NumberOfFailures               int        `json:"numberOfFailures"`
	NumberOfBeasts                 int        `json:"numberOfBeasts"`
	NumberOfEmpty              int        `json:"numberOfEmpty"`
	AvalonPower   					bool `json:"avalon_power,omitempty"`
	FinalResult                    int        `json:"finalResult"`
	Id                             float32    `json:"questId"` //e.g. 1.1 , 2 ..
	ExcaliburPlayer                string     `json:"excaliburPicker"`
	ExcaliburChosenPlayer          string     `json:"excaliburChoose"`
	LadySuggester                  string     `json:"LadySuggester"`
	LadyChosenPlayer               string     `json:"LadyChosenPlayer"`
	LadySuggesterPublishToTheWorld string     `json:"LadySuggesterPublishToTheWorld"`
}

type QuestSuggestionsManager struct {
	playersVotedYes           []string
	playersVotedNo            []string
	unsuccessfulRetries       int
	PlayerWithVeto            string
	suggesterIndex            int
	SuggestedPlayers          []string
	OnlyGoodSuggested          bool //for Meliagant
	SuggestedTemporaryPlayers string //showed until picking all quest memebers
	SuggestedCharacters       map[string]bool
	excalibur                 Excalibur
}

type CharacterName string
type Player string

type PlayerSecrets struct {
	PlayersWithSameLoyalty []string `json:"PlayersWithSameLoyalty"`
	PlayersWithDifferentLoyalty []string `json:"PlayersWithDifferentLoyalty"`
	PlayersWithGoodCharacter []string `json:"PlayersWithGoodCharacter"`
	PlayersWithBadCharacter  []string `json:"PlayersWithBadCharacter"`
	PlayersWithUncoveredCharacters  map[string]string `json:"PlayersWithUncoveredCharacters"`
	PlayerSee  string `json:"PlayersSee"`
	Seen string `json:"Seen"`
	PlayerSee2  string `json:"PlayersSee2"`
	Seen2 string `json:"Seen2"`
}

type BoardGame struct {
	whoSeeWho map[string]map[string]bool
	clientIdToPlayerName map[string]PlayerName

	numOfPlayers			int
	numOfConnectedPlayers	int
	ladyOfTheLake            LadyStats

	playersWithGoodCharacter []string //for vivian
	PlayersWithBadCharacter  []string //for vivian
	playersWithCharacters  map[string]string //for vivian

	SecretsMap				map[string]*PlayerSecrets

	Secrets                  map[string][]string
	PlayerNames              []PlayerName `json:"players,omitempty"`
	PlayerToCharacter        map[PlayerName]string
	CharacterToPlayer        map[string]PlayerName
	Characters               []string
	OtherRolesDescriptions    map[string]CharacterDescription

	PendingMurders           []Murder
	PlayerToMurderInfo       map[string]MurderInfo
	quests                   QuestManager
	archive                  []QuestArchiveItem
	lancelotCards            []int
	lancelotCardsIndex       int
	suggestions              QuestSuggestionsManager
	votesForNextMission      map[string]bool
	isSuggestionPassed       bool
	isSuggestionGood         int
	isSuggestionBad          int
	manager                  ClientManager

	QuestStage float32 // e.g. 1, 1.1, 1.2 then 2 ..
	LastQuestStage float32 // e.g. 1, 1.1, 1.2 then 2 .. if quest is canceled
	State      int     `json:"state"`
	StateDescription      string
}

var globalBoard = BoardGame{
	PlayersWithBadCharacter:  make([]string, 0),
	playersWithGoodCharacter: make([]string, 0),
	playersWithCharacters: make(map[string]string),
	clientIdToPlayerName:     make(map[string]PlayerName),
	QuestStage:               1,
	lancelotCards:            make([]int, 7),
	Secrets:                  make(map[string][]string),
	SecretsMap: 				make(map[string]*PlayerSecrets),
	PlayerToMurderInfo:       make(map[string]MurderInfo),
	quests: QuestManager{
		current:                    0,
		playersVotes:               make([][]int, 20),
		results:                    make(map[int]QuestStats),
		realResults:                make(map[int]QuestStats),
		successfulQuest:            0,
		unsuccessfulQuest:          0,
		playerVotedForCurrent:      make(map[string]int),
		playerVotedForCurrentQuest: make([]string, 0),
		differentResults:           make(map[int]int),
		Flags:                      make(map[int]bool),
	},
	archive: make([]QuestArchiveItem, 0),
	manager: ClientManager{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	},
}
