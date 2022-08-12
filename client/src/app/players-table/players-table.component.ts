import { Component, OnInit } from '@angular/core';
import {PlayersService} from "../players.service";
import {BoardGameModel} from "../model/board-game.model";
import {AuthService} from "../auth.service";

type Position = {
  left: string;
  top: string
};

@Component({
  selector: 'app-players-table',
  templateUrl: './players-table.component.html',
  styleUrls: ['./players-table.component.css']
})
export class PlayersTableComponent implements OnInit {

  board: BoardGameModel = null;
  position: Position[] = [];
  iconsPosition: Position[] = [];

  constructor(private playerService: PlayersService, public authService: AuthService) {
    this.playerService.boardSubject.subscribe((board: BoardGameModel) => {
      console.log("new board", board);
      this.board = board;
      if (!this.board) return;
      this.position = [];
      this.iconsPosition = [];
      const angle = 360 / this.board.players.all.length;
      const radius = 340;
      const iconsRadius = 270;
      let counter = 0;

      for (const player of this.board.players.all) {
        const playerAngle = counter * angle;
        this.position.push({
          left: Math.floor(radius + Math.sin(this.toRadians(playerAngle)) * radius) + 'px',
          top: Math.floor(radius - Math.cos(this.toRadians(playerAngle)) * radius) + 'px'
        })
        this.iconsPosition.push({
          left: Math.floor(iconsRadius + Math.sin(this.toRadians(playerAngle)) * iconsRadius) + 'px',
          top: Math.floor(iconsRadius - Math.cos(this.toRadians(playerAngle)) * iconsRadius) + 'px'
        })
        counter++;
      }
      console.log(angle, this.position);
    })
  }

  toRadians (angle) {
    return angle * (Math.PI / 180);
  }

  ngOnInit(): void {
  }

}
