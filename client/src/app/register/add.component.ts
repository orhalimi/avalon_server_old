import { Component, OnInit } from '@angular/core';
import { Location } from '@angular/common';
import { HttpClient } from '@angular/common/http';
import {Router} from '@angular/router';
import { PlayersService } from '../players.service';
import {SocketService} from '../socket.service';
import {FormBuilder, FormGroup, Validators} from '@angular/forms';
import {AuthService} from '../auth.service';

@Component({
  selector: 'app-add',
  templateUrl: './add.component.html',
  styleUrls: ['./add.component.css']
})
export class AddComponent implements OnInit {

  form: FormGroup;
  public player: any;
  public user: any;
  public messages: Array<any>;
  public chatBox: string;
  // tslint:disable-next-line:max-line-length
  constructor(private fb: FormBuilder, private location: Location, private router: Router, private http: HttpClient, private pl: PlayersService, private socket: SocketService, public authService: AuthService) {
    this.player = {
    Name: '',
  };
    this.user = {
      name: '',
      email: '',
    };
    this.messages = [];
    this.chatBox = '';
    this.form = this.fb.group({
      username: ['', Validators.required],
      password: ['', Validators.required]
    });
  }

  register() {
    const val = this.form.value;

    if (val.username && val.password) {
      this.authService.register(val.username, val.password);
    }
  }

  public add() {
    if (this.player.player !== '' ) {
      this.socket.send('{"type":"add_player", "player":"' + this.player.player + '"}');
      this.pl.setPlayer(this.player.player);
      this.location.back();
    }
  }

  public return() {
    this.router.navigate(['']);
  }

  public getMsg() {
    return this.pl.getMsg();
  }

  public isSystemMessage(message: string) {
    return message.startsWith("/") ? "<strong>" + message.substring(1) + "</strong>" : message;
  }

  ngOnInit(): void {
  }

}
