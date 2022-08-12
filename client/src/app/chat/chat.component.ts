import {Component, ElementRef, OnInit, ViewChild} from '@angular/core';
import {SocketService} from "../socket.service";
import {ChatMessage} from "../model/interfaces";
import {NgForm} from "@angular/forms";

@Component({
  selector: 'app-chat',
  templateUrl: './chat.component.html',
  styleUrls: ['./chat.component.css']
})
export class ChatComponent implements OnInit {

  @ViewChild('scrollMe') private myScrollContainer: ElementRef;
  messages: ChatMessage[] = [];

  constructor(private socket: SocketService) {
    this.socket.getEventListener().subscribe(event => {
      console.log("socket recive in chat", event);
      if (event.type === 'message') {
        try {
          let messageContent = JSON.parse(event.data.content);
          if (messageContent.type === 'chat_message') {
            console.log("ok it's a chat message", messageContent);
            this.messages.push({
              message: messageContent.content,
              sender: event.data.sender
            })
            setTimeout(() => {
              this.scrollToBottom();
            })
          }
        } catch (e) {
        }
      }
    })
  }

  ngOnInit(): void {
  }

  go(form: NgForm) {
    console.log("sending", form.value.pending);
    if (!form.value.pending) return;
    this.socket.send(JSON.stringify({
      "type": "chat_message",
      "content": form.value.pending
    }));
    form.setValue({
      pending: ''
    })
  }

  scrollToBottom(): void {
    try {
      this.myScrollContainer.nativeElement.scrollTop = this.myScrollContainer.nativeElement.scrollHeight;
    } catch(err) { }
  }
}
