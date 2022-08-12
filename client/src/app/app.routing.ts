import { MoviesComponent } from './movies/movies.component';
import { CreateComponent } from './create/create.component';
import { AddComponent } from './register/add.component';

export const AvailableRoutes: any = [
    { path: '', component: MoviesComponent },
    { path: 'create', component: CreateComponent },
  { path: 'add', component: AddComponent },
  { path: 'game', component: MoviesComponent }
];
