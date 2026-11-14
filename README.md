# TradeSync

Go ile yazilmis, olay tabanli bir emir eslestirme sistemi ve onun web arayuzu.
Gercek para yoktur; her hesap deneme bakiyesiyle baslar.

## Mimari

```
istemci -> gateway -> kafka(orders) -> engine -> kafka(trades, depth, order-updates)
                                                   |
                                                   v
                                               processor -> postgres (islemler, bakiyeler)
                                                         -> redis (derinlik, mumlar, acik emirler)
```

| Servis | Gorev |
| --- | --- |
| `gateway` | REST ve WebSocket. Giris, emir alma, bakiye ayirma, okuma uclari. Port 8072. |
| `engine` | Emir defterini bellekte tutar ve eslestirir. Yalniz Kafka ile konusur. |
| `processor` | Islemleri ve bakiye hareketlerini Postgres'e, anlik veriyi Redis'e yazar. |
| `marketmaker` | Piyasaya surekli emir gonderen bot. |
| `web` | React + Vite arayuzu. |
| `mobile` | Expo ile yazilmis, salt okunur defter goruntuleyici. |

## Calistirma

```bash
cp .env.example .env   # AUTH_SECRET ve MM_PASSWORD degerlerini doldurun
docker compose up -d --build
```

Gateway `http://localhost:8072` adresinde acilir. Web arayuzu icin:

```bash
cd web
echo 'VITE_API_BASE=http://localhost:8072' > .env.local
npm install
npm run dev
```

Ornek emirler gondermek icin `bash demo.sh`.

## Ayarlar

| Degisken | Servis | Aciklama |
| --- | --- | --- |
| `AUTH_SECRET` | gateway | Oturum token'larini imzalar. Bos birakilirsa her acilista yenisi uretilir ve oturumlar duser. |
| `MM_USERNAME`, `MM_PASSWORD` | gateway, marketmaker | Bot hesaplari (`<ad>` ve `<ad>-sell`). Gateway bu hesaplari acilista olusturur ve bakiye kontrolunden muaf tutar. Varsayilan sifreyi canlida kullanmayin: bu hesaplarla sinirsiz emir verilebilir. |
| `DEMO_FUNDS` | gateway | Yeni hesaba verilen deneme bakiyesi, ornegin `USD:100000,BTC:10`. |
| `CLIENT_IP_HEADER` | gateway | Vekil sunucu arkasinda gercek istemci adresini tasiyan baslik (ornegin `X-Real-IP`). Hiz siniri bu adrese gore calisir. |
| `SNAPSHOT_PATH` | engine | Emir defterinin yazildigi dosya. Bos ise goruntu alinmaz, defter her acilista `orders` topic'inin basindan kurulur. |
| `SNAPSHOT_INTERVAL` | engine | Goruntu araligi, varsayilan `30s`. |
| `KAFKA_BROKERS`, `POSTGRES_DSN`, `REDIS_ADDR` | hepsi | Altyapi adresleri. |

## API

Fiyat ve miktarlar 10^4 olcekli tamsayidir: `650000000` = 65.000,00 ve `5000` = 0,5.
Korumali uclar `Authorization: Bearer <token>` ister.

| Uc | Giris | Aciklama |
| --- | --- | --- |
| `POST /auth/register` | hayir | `{name, username, password}`; token doner. |
| `POST /auth/login` | hayir | `{username, password}`; token doner. |
| `POST /orders` | evet | `{symbol, side, price, quantity}`. Bakiye yetmezse 400. |
| `GET /orders?symbol=` | evet | Kullanicinin acik emirleri. |
| `DELETE /orders/{id}` | evet | Iptal istegi. Asenkrondur. |
| `GET /balances` | evet | Varlik basina kullanilabilir ve emirde bekleyen tutar. |
| `GET /trades?symbol=&limit=` | hayir | Son islemler. Kullanici ve emir kimligi icermez. |
| `GET /trades/mine?symbol=&limit=` | evet | Kullanicinin kendi islemleri, yon bilgisiyle. |
| `GET /depth?symbol=` | hayir | Emir defterinin ilk 15 kademesi. |
| `GET /klines?symbol=&limit=` | hayir | Dakikalik mumlar; varsayilan son 1440. |
| `GET /ws` | hayir | Derinlik ve islem akisi. |

## Davranis notlari

- **Bakiye:** Emir engine'e gitmeden once karsiligi ayrilir (alis icin fiyat x miktar, satis icin miktar).
  Islem limitten ucuza gerceklesirse fark iade edilir; iptalde kalan tutar serbest kalir.
- **Kendi emriyle eslesme:** Gelen emir kullanicinin kendi bekleyen emrine denk gelirse kalan kismi iptal edilir.
- **Tekrar guvenligi:** Her islemin engine'in verdigi sabit bir kimligi vardir. Ayni islem iki kez gelse de
  Postgres'e bir kez yazilir ve bakiyeye bir kez yansir.
- **Yeniden baslatma:** Engine defteri diskteki goruntuden kurar, sonra `orders` topic'inde kaldigi yerden devam eder.
  Daha once yayinlanmis sonuclar tekrar yayinlanmaz.
- **Hiz siniri:** Ayni adresten 5 dakikada 10 hatali giris ya da saatte 10 kayit sonrasi 429 doner.
  Sayaclar gateway belleginde tutulur.

## Gelistirme

```bash
go vet ./engine/... ./gateway/... ./processor/... ./marketmaker/... ./shared/...
go test -race ./engine/... ./gateway/... ./processor/... ./marketmaker/... ./shared/...
cd web && npm run lint && npm run build
```

Veritabani testleri `TEST_POSTGRES_DSN` tanimliysa calisir, degilse atlanir:

```bash
docker run -d --rm --name ts-test-pg -e POSTGRES_PASSWORD=test -e POSTGRES_DB=test -p 127.0.0.1:55432:5432 postgres:16
TEST_POSTGRES_DSN=postgres://postgres:test@127.0.0.1:55432/test go test ./gateway/... ./processor/...
```

Sema `shared/schema/schema.sql` dosyasindadir; gateway ve processor acilista uygular.

## Bilinen sinirlar

- Para yatirma ve cekme yoktur; bakiyeler yalniz deneme bakiyesi ve islemlerle degisir.
- Kafka yazimi basarisiz sayilip emir yine de engine'e ulasirsa o emrin ayrilan tutari geri verilmis olur.
- Gateway bakiyeyi ayirdiktan sonra Kafka'ya yazamadan cokerse tutar emirde bekliyor gorunur.
- Mobil uygulama emir gondermez.
