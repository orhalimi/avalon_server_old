import {Component, OnInit, OnDestroy, ViewEncapsulation} from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router } from '@angular/router';
import { Location } from '@angular/common';
import { PlayersService } from '../players.service';
import {interval, pipe} from 'rxjs';
import { SocketService } from '../socket.service';
import { TruncatePipe } from '../limitpipe';
import { RoundPipe } from '../rountpipe';
import { AuthService } from '../auth.service';
import {FormBuilder, FormGroup, Validators} from '@angular/forms';

@Component({
    selector: 'app-movies',
    templateUrl: './movies.component.html',
    styleUrls: ['./movies.component.css']
})


export class MoviesComponent implements OnInit {

  form: FormGroup;
  public messages: Array<any>;
  public filterqu: Array<boolean>;
  public chatBox: string;
  public errMsg: string;
  maxNo = false;
  MurdermaxNo = false;
  amt = 0;
    public movies: any;
    public players: any;
  public player: any;
  public state: any;
  public vote: any;
  public votes: any;
  public journeyVote: any;
  public secrets: any;
  public board: any;
  public flipped: any;
  public collapsed: any;
  public myClass: any;
  public suggestion: any;
  public user: any;
  public oldState: any;
  public panelOpenState: any;
  public showSuggestion: boolean;
  public showPlayer: any;
  public results: any;
  public playersSuggestion: any;
  public colorLabels: any;
  public assassinKill: any;
  public excaliburPick: any;
  public ExcaliburAmt: any;
  public ladyPick: any;
  public ladyResponse: any;
  public assassinKillErrorMsg: any;
  // tslint:disable-next-line:max-line-length
  public constructor(private fb: FormBuilder, private http: HttpClient, private router: Router, private location: Location, public pl: PlayersService, private socket: SocketService, public authService: AuthService) {
        this.movies = [];
        this.colorLabels = {
      Merlin: 'Blue',
      green: 'Green',
      red: 'Red',
    };
        this.players = [];
        this.vote = [];
        this.user = [];
        this.state = 0;
        this.votes = [];
        this.journeyVote = [];
        this.secrets = [];
        this.maxNo = false;
        this.MurdermaxNo = false;
        this.amt = 0;
        this.ExcaliburAmt = 0;
        this.board = [];
        this.suggestion = [];
        this.oldState = 0;
        this.results = [];
        this.myClass = 'xxssw';
        this.showPlayer = '';
        this.showSuggestion = false;
        this.messages = [];
        this.excaliburPick = [];
        this.playersSuggestion = new Set();
        this.filterqu = [true, true, true, true, true, true, true, true, true];
        this.chatBox = '';
        this.errMsg = '';
        this.assassinKill = '';
        this.assassinKillErrorMsg = '';
        this.ladyPick = '';
        this.ladyResponse = 0;
        this.journeyVote.vote = -1;
        this.form = this.fb.group({
        username: ['', Validators.required],
        password: ['', Validators.required]
      });
    }

    public ngOnInit() {
        this.location.subscribe(() => {
            this.refresh();
        });
        document.body.classList.add('bg-img');
        this.amt = 0;
        this.refresh();
    }

  // tslint:disable-next-line:use-lifecycle-interface
  public ngOnDestroy() {
    console.log('ChildComponent:OnDestroy');
  }

  logout() {
      this.authService.logout();
      this.socket.close();
  }

  ShowPlayer(pl: string) {
    this.showPlayer = pl;
  }

  login() {
    const val = this.form.value;

    if (val.username && val.password) {
      // tslint:disable-next-line:no-unused-expression
      this.authService.login(val.username, val.password, () => { this.socket.openSocket; });
    }
  }

  onChange(player: string, isChecked: boolean) {
    if (isChecked) {
      this.amt++;
      this.errMsg = '';
      console.log('amt==', this.amt);
      this.playersSuggestion.add(player);
      this.socket.send('{"type":"suggestion_tmp", "content":' + JSON.stringify([...this.playersSuggestion]) + '}');
      console.log('playersSuggestion==',  this.playersSuggestion);
    }
    else {
      this.amt--;
      this.errMsg = '';
      console.log('amt2==', this.amt);
      this.playersSuggestion.delete(player);
      this.socket.send('{"type":"suggestion_tmp", "content":' + JSON.stringify([...this.playersSuggestion]) + '}');
    }
    console.log('amt3=', this.pl.boardGame.results[this.pl.boardGame.current].numofplayers);
    this.amt === this.pl.boardGame.results[this.pl.boardGame.current].numofplayers ? this.maxNo = true : this.maxNo = false;
  }

  // tslint:disable-next-line:typedef
  counter(i: number) {
    return new Array(i);
  }

  // tslint:disable-next-line:typedef
  onChangeExcaliburSuggest(player: string, isChecked: boolean) {
    if (isChecked) {
      this.ExcaliburAmt++;
      this.errMsg = '';
      this.excaliburPick = player;
    }
    else {
      this.ExcaliburAmt--;
      this.errMsg = '';
    }
  }

  onChangeExcalibur(player: string, isChecked: boolean) {
    if (isChecked) {
      this.amt++;
      this.errMsg = '';
      console.log('ex_or==', this.amt);
      this.playersSuggestion.add(player);
    }
    else {
      this.amt--;
      this.errMsg = '';
      console.log('ex2==', this.amt);
      this.playersSuggestion.delete(player);
    }
    this.amt == 1 ? this.maxNo = true : this.maxNo = false;
  }

    private refresh() {
    }

  public getMsg() {
    return this.pl.getMsg();
  }

  public isSystemMessage(message: string) {
    return message != null && message.startsWith('/') ? '<strong>' + message.substring(1) + '</strong>' : message;
  }

  public send() {
    if (this.chatBox) {
      this.socket.send(this.chatBox);
      this.chatBox = '';
    }
  }

  public FilterQuests(index) {
      const newIndex = Math.floor(index);
      this.filterqu[newIndex] = !this.filterqu[newIndex];
  }

  public LadySuggest(suggestion: string) {
    // tslint:disable-next-line:max-line-length
    this.socket.send('{"type":"lady_suggest", "content": "' + suggestion + '"}');
    console.log(suggestion);
    this.ladyPick = '';
  }

  public LadyResponse(response: string) {
    // tslint:disable-next-line:max-line-length
  if (response === 'Good') {
    this.ladyResponse = 1;
  } else {
    this.ladyResponse = 0;
  }

  this.socket.send('{"type":"lady_response", "content": '
      + JSON.stringify(this.ladyResponse) + '}');
  console.log(this.ladyResponse);
  this.ladyPick = '';
  }


  public LadyPublish(publish: string) {
    // tslint:disable-next-line:max-line-length
    if (publish === 'Good') {
      this.ladyResponse = 1;
    } else {
      this.ladyResponse = 0;
    }

    this.socket.send('{"type":"lady_publish_response", "content": '
      + JSON.stringify(this.ladyResponse) + '}');
    console.log(this.ladyResponse);
    this.ladyPick = '';
    this.ladyResponse = 0;
  }


  public Suggest() {
    if (this.pl.boardGame.excalibur) {
      if (this.ExcaliburAmt > 1) {
        this.errMsg = 'Error! Only 1 player can get the excalibur.';
        return;
      }
      if (this.ExcaliburAmt === 0) {
        this.errMsg = 'Error! Please choose one player!';
        return;
      }
      this.ExcaliburAmt = 0;
    }
    this.errMsg = '';
    this.amt = 0;
    // tslint:disable-next-line:max-line-length
    this.socket.send('{"type":"suggestion", "content": {"players" : ' + JSON.stringify([...this.playersSuggestion]) + ', "excalibur": ' + JSON.stringify(this.excaliburPick) + '}}');
    /*this.http.post('http://localhost:12345/suggestion', JSON.stringify(this.players))
        .subscribe(result => {
          this.suggestion.voted == true;
        });*/
    console.log(this.playersSuggestion);
    this.playersSuggestion = new Set();
    this.suggestion.voted = true;
    this.maxNo = false;
    this.excaliburPick = [];
  }

  public Pick() {
    this.amt = 0;
    // tslint:disable-next-line:max-line-length
    this.socket.send('{"type":"excalibur_pick", "content":' + JSON.stringify([...this.playersSuggestion]) + '}');

    console.log(this.playersSuggestion);
    this.playersSuggestion = new Set();
    this.suggestion.voted = true;
    this.maxNo = false;
    this.excaliburPick = [];
  }

  public Murder() {
    this.amt = 0;
    if (this.assassinKill === '' && this.pl.boardGame.murder.byCharacter === 'Assassin') {
      this.assassinKillErrorMsg = 'You must choose character!';
      return;
    }
    // tslint:disable-next-line:max-line-length
    this.socket.send('{"type":"murder", "content": { "assassinkill": "' + this.assassinKill + '", "rest":' + JSON.stringify(this.pl.boardGame.players.all) + '}}');
    this.MurdermaxNo = false;
    this.assassinKill = '';
    this.assassinKillErrorMsg = '';
  }

  public SirPick() {
    this.amt = 0;
    // tslint:disable-next-line:max-line-length
    this.socket.send('{"type":"sir_pick", "content": { "pick": "' + this.assassinKill + '"}}');
    this.assassinKill = '';
  }

  public reset() {
    this.socket.send('{"type":"reset", "content":""}');
  }

  public SendGoodVote() {
    this.player = this.authService.name;
    this.vote.vote = true;
    this.pl.showSuggestion = false;
    if (this.authService.name !== '' ) {
      this.socket.send('{"type":"vote_for_suggestion", "content":' + '{"playerName": "' + this.player + '", "vote": true }' + '}');
    }
  }

  public SendBadVote() {
    this.player = this.authService.name;
    this.pl.showSuggestion = false;
    if (this.authService.name !== '' ) {
      this.socket.send('{"type":"vote_for_suggestion", "content":' + '{"playerName": "' + this.player + '", "vote": false }' + '}');
    }
  }

  public JourneyVote() {
    this.player = this.authService.name;
    if (this.journeyVote.vote < 0) { return; }
    if (this.player !== '' ) {
      // tslint:disable-next-line:max-line-length
      this.socket.send('{"type":"vote_for_journey", "content":' + '{"playerName": "' + this.player + '", "vote": ' + this.journeyVote.vote + ' }' + '}');
      this.journeyVote = [];
      this.journeyVote.vote = -1;
      this.journeyVote.errorCode = null;
    }
  }

    public create() {
        this.router.navigate(['create']);
    }

  public add() {
    this.router.navigate(['add']);
  }
}
