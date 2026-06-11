# TradeSync Pro - Mevcut İlerleme ve Mimari Durumu

## Proje Amacı
Go (Golang) kullanılarak saf **Fonksiyonel Programlama (FP)** prensipleriyle yazılmış, ultra düşük gecikmeli (low-latency) bir **High Frequency Trading (HFT) Eşleştirme Motoru** ve tam teşekküllü modern bir borsa arayüzü geliştirmek.

## 🏗️ Sistem Mimarisi (Event-Driven & CQRS)

Sistem, yazma (Command) ve okuma (Query) işlemlerini birbirinden ayıracak şekilde 3 ana Go mikroservisinden ve bir mesaj kuyruğundan oluşmaktadır:

1. **Gateway (Port 8072):**
   - Dış dünya ile iletişim kuran REST API ve WebSocket sunucusu.
   - Emirleri (Orders) alır ve doğrudan Kafka `orders` topic'ine fırlatır.
   - React arayüzüne derinlik (`/depth`), mum grafikleri (`/klines`) ve geçmiş işlemleri (`/trades`) sunar.

2. **Kafka (Mesaj Kuyruğu):**
   - Servisler arası asenkron, kayıpsız iletişim sağlar. (`orders`, `trades`, `depth` topic'leri)

3. **Engine (Eşleştirme Motoru):**
   - **(Kalp)** Sadece Kafka'dan emir okur, eşleştirir ve sonucu Kafka'ya `trades` olarak atar.
   - **(FP Prensibi)** Veritabanına bağlanmaz, HTTP isteği atmaz. Sadece Girdi (Order) -> Çıktı (Trade) mantığıyla çalışan "Saf (Pure)" bir servistir.
   - Emir defteri (Orderbook) bellek (RAM) üzerinde tutulur. Durum değiştikçe Redis'e anlık "Snapshot" atar.

4. **Processor:**
   - Eşleşen işlemleri (Trades) Kafka'dan okur.
   - Geçmişi kalıcı olarak **PostgreSQL**'e kaydeder.
   - Her işlemden anında **Candlestick (OHLC)** mum grafiği matematiğini hesaplar ve **Redis**'e saniyesinde yazar.

## 💻 Frontend (React & Vite)

- **UI/UX:** Karanlık mod (Dark Mode) ve "Quant" stili, yoğun veri akışına uygun profesyonel tasarım (`#121212` arka plan, JetBrains Mono fontlar).
- **Gerçek Zamanlı Tahta (Orderbook):** Kırmızı (Ask) ve Yeşil (Bid) emirlerin derinlik çubuklarıyla (Depth Bars) canlı gösterimi.
- **TradingView Grafiği:** `lightweight-charts` v5 kütüphanesi entegre edildi. Fiyat hareketleri anlık olarak yeşil/kırmızı mumlarla (Candlestick) çiziliyor.
- **Akıllı Emir Formu:** 
  - Limit/Market/Stop karmaşası giderildi, devasa AL ve SAT butonları konuldu.
  - AL ve SAT'a tıklandığında tahtadaki en iyi fiyatı (Best Ask / Best Bid) otomatik doldurma eklendi.
  - %25, %50, %75, %100 hesaplama butonları aktif edildi.
- **Kullanıcı ve Cüzdan Sistemi:** 
  - Sağ üstte DiceBear API ile anlık Avatar üreten Profil Menüsü.
  - Hover/Tıklama ile açılan "Cüzdan Özeti" dropdown'u (Tether, Bitcoin, Ethereum bakiyeleri gösterimi).

## 🚀 Sonraki Adım
Sistemin kodlaması, arayüzü ve entegrasyonu tamamen bitti. Üniversite sunumu için Engine katmanında **"Fonksiyonel Programlama (FP)"** kurallarının nasıl uygulandığını (Saf fonksiyonlar, Side-effect izolasyonu vb.) anlatan akademik ve teknik **Raporun / README.md'nin** yazılması.
