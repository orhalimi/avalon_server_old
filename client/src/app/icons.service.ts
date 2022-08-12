import { Injectable } from '@angular/core';
import {MatIconRegistry} from "@angular/material/icon";
import {DomSanitizer} from "@angular/platform-browser";

@Injectable({
  providedIn: 'root'
})
export class IconsService {

  constructor(iconRegistry: MatIconRegistry, sanitizer: DomSanitizer) {
    iconRegistry.addSvgIcon('sword', sanitizer.bypassSecurityTrustResourceUrl('assets/svg-icons/sword.svg'));
    iconRegistry.addSvgIcon('lady', sanitizer.bypassSecurityTrustResourceUrl('assets/svg-icons/lady.svg'));
  }
}
