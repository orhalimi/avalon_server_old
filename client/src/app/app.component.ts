import { Component, ViewEncapsulation } from '@angular/core';
import {IconsService} from "./icons.service";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.css'],
  encapsulation: ViewEncapsulation.None
})
export class AppComponent {
  title = 'blabla';

  constructor(public iconService: IconsService) {
  }
}
