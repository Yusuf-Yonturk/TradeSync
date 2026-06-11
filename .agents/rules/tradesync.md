# TradeSync Workspace Rules

## Proje bağlamı

Bu proje bir HFT Order Matching Engine projesi.
Adı: TradeSync

Amaç:
- alış/satış emirlerini düşük gecikmeyle eşleştirmek
- price-time priority korumak
- event-driven mimari kurmak
- sonradan replay edilebilir yapı hazırlamak

Bu proje Fonksiyonel Programlama dersi için geliştiriliyor.
Aynı backend daha sonra web ve mobil istemcilerle de uyumlu çalışmalı.

## Teknoloji yığını

- Go
- Kafka
- PostgreSQL
- Redis
- Flutter
- WebSocket + Protobuf

Ama şu an öncelik altyapı değil, engine domain katmanı.

## Mimari

Client
-> Gateway
-> Kafka (orders)
-> Engine
-> Kafka (trades)
-> Processor
-> Redis + PostgreSQL

Servis sorumlulukları:
- Gateway: bağlantı, validation, message publish
- Engine: order matching core
- Processor: read model, persistence

Şu an odak:
- engine/orderbook/order.go
- engine/orderbook/heap.go
- engine/orderbook/orderbook.go
- matching algoritması
- unit testler

Kafka, Redis, PostgreSQL, WebSocket katmanlarını engine domain içine erken sokma.

## Domain kuralları

Order modeli en az şunları düşünmeli:
- ID
- UserID
- Symbol
- Side
- Price
- Quantity
- Remaining
- CreatedAt

Matching kuralları:
- Buy tarafı en yüksek fiyat öncelikli
- Sell tarafı en düşük fiyat öncelikli
- Aynı fiyat seviyesinde önce gelen önce eşleşir
- Partial fill desteklenmeli
- Tam dolan emir book’tan düşmeli
- Uygun olduğunda trade verisi üretilebilmeli

Veri yapısı:
- Buy side -> max-heap
- Sell side -> min-heap

Heap veya orderbook tasarımı price-time priority’yi bozmamalı.

## Geliştirme sırası

Adımları gereksiz yere karıştırma.

Öncelik sırası genel olarak:
1. order modeli
2. heap yapısı
3. orderbook yapısı
4. matching mantığı
5. unit testler
6. entegrasyon katmanları

Bir adım bitmeden sonraki büyük katmana geçme.

## Kodlama kuralları

- Go 1.22 uyumlu yaz
- Gereksiz üçüncü parti paket kullanma
- Standard library öncelikli olsun
- Gereksiz interface yazma
- Gereksiz generic yazma
- Domain mantığını mümkün olduğunca sade tut
- Test edilebilirlik yüksek olsun
- Yan etkileri mümkünse çekirdek mantığın dışında tut

## Fonksiyonel programlama perspektifi

Bu projede şunları görünür kıl:
- saf fonksiyonlar
- anlaşılır veri akışı
- test edilebilir mantık
- yan etkilerin ayrılması

Ama Go’ya aykırı yapay bir fonksiyonellik kurma.
Pratik ve anlaşılır çözüm tercih et.

## Cevap formatı

Her adımda bu akışı izle:
1. Bu adımda ne yapıyoruz?
2. Neden şimdi bunu yapıyoruz?
3. Kod
4. Test
5. Sıradaki adım ne?
6. Anladın mı, devam edelim mi?

## Yorum stili

Kod yorumları:
- kısa
- az
- doğal

Uzun, öğretici paragraf gibi AI yorumları yazma.

## Kritik davranış

- Bütün sistemi bir anda yazma
- Adım adım ilerle
- Önce neden, sonra kod, sonra test
- Sade Türkçe kullan
- 3. sınıf yazılım öğrencisine anlatır gibi konuş