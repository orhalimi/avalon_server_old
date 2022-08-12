export interface All {
  player: string;
}

export interface Active {
  player: string;
}

export interface Players {
  total: number;
  all: All[];
  active: Active[];
}

export interface Galahad {
  loyalty: string;
  canVote: string[];
  canVoteOnFlush: string[];
}

export interface Nimue {
  loyalty: string;
  canSee: string;
  canVote: string[];
  canVoteOnFlush: string[];
}

export interface Oberon {
  loyalty: string;
  canVote: string[];
  canVoteOnFlush: string[];
  murderedBy: string;
  specialRole: string;
}

export interface Titanya {
  loyalty: string;
  canVote: string[];
  canVoteOnFlush: string[];
}

export interface Characters {
  Galahad: Galahad;
  Nimue: Nimue;
  Oberon: Oberon;
  Titanya: Titanya;
}

export interface Suggester {
  player: string;
}

export interface Archive {
  playersAcceptedQuest: string[];
  playersNotAcceptedQuest?: any;
  suggester: Suggester;
  suggestedPlayers: string[];
  isSuggestionAccepted: boolean;
  isSuggestionOver: boolean;
  switch: boolean;
  numberOfReversal: number;
  numberOfSuccesses: number;
  numberOfFailures: number;
  numberOfBeasts: number;
  finalResult: number;
  questId: number;
  excaliburPicker: string;
  excaliburChoose: string;
  LadySuggester: string;
  LadyChosenPlayer: string;
  LadySuggesterPublishToTheWorld: string;
}

export interface Secrets {
  character: string;
}

export interface PlayerSecret {
  PlayersWithSameLoyalty: string[];
  PlayersWithDifferentLoyalty: string[];
  PlayersWithGoodCharacter: string[];
  PlayersWithBadCharacter: string[];
  PlayersWithUncoveredCharacters: { [key: string]: string };
  PlayersSee: string;
  Seen: string;
  PlayersSee2: string;
  Seen2: string;
}

export interface Murder {
  target?: any;
  by: string;
  byCharacter: string;
  StateAfterSuccess: number;
}

export interface Sir {
}

export interface Result {
  final?: number;
  ppp?: number;
  numofplayers?: number;
  successes?: number;
  reversals?: number;
  failures?: number;
  beasts?: number;
  avalon_power?: boolean;
  empty?: number;
}

export interface PlayerInfo {
  ch?: string;
  isKilled?: boolean;
}



export interface BoardGameModel {
  players: Players;
  current: number;
  active_players_num: number;
  characters: Characters;
  size: number;
  state: number;
  stateDescription: string;
  archive: Archive[];
  secrets: Secrets;
  playerSecrets: PlayerSecret;
  suggester: string;
  murder: Murder;
  sir: Sir;
  optionalVotes: string[];
  suggesterVeto: string;
  onlyGoodSuggested: boolean;
  suggestedPlayers: string[];
  suggestedTemporaryPlayers: string;
  PlayersVotedForCurrQuest: string[];
  PlayersVotedYesForSuggestion: string[];
  PlayersVotedNoForSuggestion: string[];
  results: { [key: string]: Result };
  playerToCharacters: { [key: string]: PlayerInfo };
  excalibur: boolean;
  suggestedExcalibur: string;
  isLady: boolean;
  ladySuggester: string;
  ladyChosenPlayer: string;
  ladyResponse: string;
  LadyResponseOptions: string[];
  ladyPublish: string;
  ladyPreviousSuggester: string;
}



