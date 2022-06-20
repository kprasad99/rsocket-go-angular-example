import { Component, OnInit } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { RsocketService, StringCodec, JsonCodec } from './rsocket.service';
import { switchMap } from 'rxjs';
import { ThisReceiver } from '@angular/compiler';
import { RSocketRequester } from 'rsocket-messaging';
import { RxRequestersFactory } from 'rsocket-adapter-rxjs';

@Component({
  selector: 'kp-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
})
export class AppComponent implements OnInit {
  title = 'rsocket-angular-ui';

  tabs = new Map<string, any[]>();

  constructor(private http: HttpClient, private rsoc: RsocketService) {}

  ngOnInit(): void {}

  generate(): void {
    let params = new HttpParams();
    const counts = 10;
    params = params.append('tabs', counts);
    this.http.get<string[]>('/api/trigger', {
      params
    }).subscribe((v) => {
      this.tabs = new Map<string, any[]>();
      for (const s of v) {
        this.tabs.set(s, []);
        this.rsoc
          .requester()
          .route('')
          .request(
            RxRequestersFactory.requestStream(
              s,
              new StringCodec(),
              new JsonCodec<any>()
            )
          )
          .subscribe((r) => {
            const data = this.tabs.get(r.id);
            if(data) {
              data.push(r.data);
            }
          });
      }
    });
  }
}
