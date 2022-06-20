import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { RSocket, RSocketConnector } from 'rsocket-core';
import { Codec, RSocketRequester } from 'rsocket-messaging';
import { WebsocketClientTransport } from 'rsocket-websocket-client';
import { firstValueFrom, switchMap, tap } from 'rxjs';
import { environment } from 'src/environments/environment';

export class StringCodec implements Codec<string> {
  readonly mimeType: string = "text/plain";

  decode(buffer: Buffer): string {
    return buffer.toString();
  }
  encode(entity: string): Buffer {
    return Buffer.from(entity);
  }
}

export class JsonCodec<T> implements Codec<T> {
  readonly mimeType: string = "application/json";

  decode(buffer: Buffer): T {
    return JSON.parse(buffer.toString());
  }
  encode(entity: T): Buffer {
    return Buffer.from(JSON.stringify(entity));
  }
}


@Injectable()
export class RsocketService {
  errorMsg = '';
  rsocket!: RSocket;

  constructor(private http: HttpClient) {}

  async init(): Promise<any> {
    if (environment.wsUrl) {
      const connector = new RSocketConnector({
        transport: new WebsocketClientTransport({
          url: environment.wsUrl,
          wsCreator: (url) => new WebSocket(url) as any,
        }),
      });
      const x = await connector.connect();
      this.rsocket = x;
      return this.rsocket;
    } else {
      return firstValueFrom(
        this.http.get<any>('/api/configuration').pipe(
          switchMap((x) => {
            const connector = new RSocketConnector({
              transport: new WebsocketClientTransport({
                url: `ws://${window.location.hostname}:${x.rsocketPort}`,
                wsCreator: (url) => new WebSocket(url) as any,
              }),
            });
            return connector.connect();
          }),
          tap((x) => (this.rsocket = x))
        )
      );
    }
  }

  ngOnDestroy(): void {
    if (this.rsocket) {
      this.rsocket.close();
    }
  }

  public requester(): RSocketRequester {
    return RSocketRequester.wrap(this.rsocket);
  }
}
