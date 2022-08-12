import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';

import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import {HttpClientModule, HttpClient, HTTP_INTERCEPTORS} from '@angular/common/http';
import {CanActivate, RouterModule} from '@angular/router';

import { MatInputModule } from '@angular/material/input';
import { AvailableRoutes } from './app.routing';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { CreateComponent } from './create/create.component';
import { MoviesComponent } from './movies/movies.component';
import { MaterialModule } from './my-material.module';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { AddComponent } from './register/add.component';
import { VotesComponent } from './votes/votes.component';
import { MatRadioModule } from '@angular/material/radio';

import { MatSliderModule } from '@angular/material/slider';
import { MatExpansionModule } from '@angular/material/expansion';
import { SocketService } from './socket.service';
import { DialogComponent } from './dialog/dialog.component';

import { TruncatePipe } from './limitpipe';
import {RoundPipe} from './rountpipe';
import {AuthService} from './auth.service';
import { AuthInterceptorService } from './auth-interceptor.service';
import { AuthGuardService } from './auth-guard.service';
import { PlayersTableComponent } from './players-table/players-table.component';
import {FlexLayoutModule} from '@angular/flex-layout';
import { QuestsOverviewComponent } from './quests-overview/quests-overview.component';
import { ChatComponent } from './chat/chat.component';

import {MatTabsModule} from '@angular/material/tabs';

@NgModule({
  declarations: [
    AppComponent,
    CreateComponent,
    MoviesComponent,
    AddComponent,
    VotesComponent,
    DialogComponent,
    TruncatePipe,
    RoundPipe,
    PlayersTableComponent,
    QuestsOverviewComponent,
    ChatComponent
  ],
    imports: [
        BrowserModule,
        AppRoutingModule,
        MatTabsModule,
        FormsModule,
        HttpClientModule,
        RouterModule,
        RouterModule.forRoot(AvailableRoutes),
        BrowserAnimationsModule,
        MaterialModule,
        MatInputModule,
        MatRadioModule,
        MatSliderModule,
        MatExpansionModule,
        ReactiveFormsModule,
      FlexLayoutModule
    ],
  providers: [SocketService,
    AuthService,
    {
      provide: HTTP_INTERCEPTORS,
      useClass: AuthInterceptorService,
      multi: true
    },
    AuthGuardService],
  bootstrap: [AppComponent]
})
export class AppModule {
}
