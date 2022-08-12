import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import {BehaviorSubject, Observable} from 'rxjs';
import { interval } from 'rxjs';
import {SocketService} from './socket.service';
import {BoardGameModel} from "./model/board-game.model";

@Injectable({
  providedIn: 'root'
})
export class PlayersService {

  boardSubject = new BehaviorSubject<BoardGameModel>(null);
  public messages: Array<any>;
  public chatBox: string;
  public gameState: any;
  public bla: any;
  public json: any;
  public boardGame: any;
  public numOfPlayers: any;
  public oldState: any;
  public showSuggestion: boolean;
  // tslint:disable-next-line:variable-name
  constructor(private socket: SocketService) {
    // Create observable stream to output our data
    this.bla = [];
    this.boardGame = [];
    this.numOfPlayers = 0;
    this.messages = [];
    this.chatBox = '';
    this.showSuggestion = false;
    this.oldState = 0;
    this.socket.getEventListener().subscribe(event => {
      if (event.type === 'message') {
        if (event.data.ty && event.data.ty === 'bla' && this.boardGame.state === 0) {
          if (event.data.players) {
            // this.bla = JSON.parse(data);
            console.log('bug');
            this.boardGame.players = event.data.players;
            this.boardSubject.next(this.boardGame);
          }
        } else {
          let data = event.data.content;
          if (this.isJson(data)) {
            let messageContent = JSON.parse(data);
            if (messageContent.type !== 'chat_message') {
              this.boardGame = messageContent;
              this.boardSubject.next(this.boardGame);
              if (this.oldState !== 3 && this.boardGame.state === 3) {
                this.showSuggestion = true;
              }
              this.oldState = this.boardGame.state;
              data = 'aaa';
            }
          }
          if (event.data.sender) {
              data = event.data.sender + ': ' + data;
            }
          this.messages.push(data);
        }

      }
      if (event.type === 'close') {
        this.messages.push('/The socket connection has been closed');
        this.boardGame = [];
        this.boardSubject.next(this.boardGame);
      }
      if (event.type === 'open') {
        this.messages.push('/The socket connection has been established');
        this.socket.send('{"type":"refresh", "content":""}');
        console.log('bla');
      }
    });
  }

  public isJson(str) {
    try {
      JSON.parse(str);
    } catch (e) {
      return false;
    }
    return true;
  }

  public isSystemMessage(message: string) {
    return message.startsWith('/') ? '<strong>' + message.substring(1) + '</strong>' : message;
  }
  getMsg(): any {
    return this.messages;
  }
  setMsg(pl: any): void {
    this.messages.push(pl);
  }
  getPlayer(): any {
    return this.bla;
  }
 setPlayer(pl: any): void {
    this.bla = pl;
  }
  setNumOfPlayers(pl: any): void {
    this.numOfPlayers = pl;
  }
  incByOne(): void {
    this.numOfPlayers++;
  }
  getNumOfPlayers(): any {
    return this.numOfPlayers;
  }
}
