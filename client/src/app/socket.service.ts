import { Injectable, EventEmitter } from '@angular/core';
import { AuthService } from './auth.service';

@Injectable()
export class SocketService {

  private socket: WebSocket;
  private listener: EventEmitter<any> = new EventEmitter();

  public constructor(private authService: AuthService) {
    this.authService.authenticatedSubject.subscribe(isAuthenticated => {
      if (isAuthenticated) {
        this.openSocket();
      }
    })
  }


  openSocket() {
    this.socket = new WebSocket('ws://127.0.0.1:8080/ws?token=' + this.authService.token); //localhost
    // this.socket = new WebSocket('ws://3.121.195.232:12345/ws?token=' + this.authService.token); //localhost
    this.socket.onopen = event => {
      this.listener.emit({ type: 'open', data: event });
    }
    this.socket.onclose = event => {
      this.listener.emit({ type: 'close', data: event });
    }
    this.socket.onmessage = event => {
      this.listener.emit({ type: 'message', data: JSON.parse(event.data) });
    };
  }

  public send(data: string) {
    this.socket.send(data);
  }

  public close() {
    this.socket.close();
  }

  public getEventListener() {
    return this.listener;
  }

}
