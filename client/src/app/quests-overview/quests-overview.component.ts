import { Component, OnInit } from '@angular/core';
import {PlayersService} from "../players.service";
import {AuthService} from "../auth.service";
import {BoardGameModel} from "../model/board-game.model";

@Component({
  selector: 'app-quests-overview',
  templateUrl: './quests-overview.component.html',
  styleUrls: ['./quests-overview.component.css']
})
export class QuestsOverviewComponent implements OnInit {
  board: BoardGameModel = null;

  constructor(private playerService: PlayersService) {
    this.playerService.boardSubject.subscribe((board: BoardGameModel) => {
      this.board = board;
    })
  }
  ngOnInit(): void {
  }

}
