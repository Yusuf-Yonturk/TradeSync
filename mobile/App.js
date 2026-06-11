import React, { useEffect, useState } from 'react';
import { StyleSheet, Text, View, ScrollView, TouchableOpacity, TextInput, SafeAreaView } from 'react-native';
import { fetchDepth, fetchTrades } from './src/api';

const WS_URL = "wss://715b27a221be60.lhr.life/ws";

export default function App() {
  const [market] = useState("BTC-USD");
  const [depth, setDepth] = useState({ bids: [], asks: [] });
  const [trades, setTrades] = useState([]);
  const [lastPrice, setLastPrice] = useState(null);
  
  useEffect(() => {
    // Initial fetch
    fetchDepth(market).then(setDepth).catch(console.error);
    fetchTrades(market).then(t => {
      setTrades(t);
      if (t.length > 0) setLastPrice(t[0].price);
    }).catch(console.error);

    // WebSocket connection
    const ws = new WebSocket(WS_URL);
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.event_type === "depth_update" && msg.symbol === market) {
          setDepth({ bids: msg.bids || [], asks: msg.asks || [] });
        } else if (msg.event_type === "trade_update" && msg.symbol === market) {
          setTrades(prev => [msg, ...prev].slice(0, 50));
          setLastPrice(msg.price);
        }
      } catch (err) {
        console.log("ws parse error", err);
      }
    };

    return () => ws.close();
  }, [market]);

  const formatPrice = (p) => Number(p / 10000).toLocaleString("tr-TR", { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  const formatQty = (q) => Number(q / 10000).toLocaleString("tr-TR", { maximumFractionDigits: 4 });

  const maxAsk = Math.max(...(depth.asks || []).map(a => a.quantity), 1);
  const maxBid = Math.max(...(depth.bids || []).map(b => b.quantity), 1);

  return (
    <SafeAreaView style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>TradeSync <Text style={styles.proBadge}>PRO</Text></Text>
        <Text style={styles.marketText}>{market}</Text>
      </View>

      <ScrollView style={styles.content}>
        {/* Price Display */}
        <View style={styles.priceContainer}>
          <Text style={[styles.bigPrice, { color: '#02c076' }]}>
            {lastPrice ? formatPrice(lastPrice) : '---'}
          </Text>
        </View>

        {/* Orderbook */}
        <View style={styles.panel}>
          <Text style={styles.panelTitle}>Emir Defteri</Text>
          <View style={styles.obHeader}>
            <Text style={styles.obHeaderText}>Fiyat</Text>
            <Text style={styles.obHeaderText}>Adet</Text>
          </View>
          
          <View style={styles.asksContainer}>
            {[...(depth.asks || [])].reverse().map((a, i) => (
              <View key={`ask-${i}`} style={styles.obRow}>
                <View style={[styles.depthBar, { backgroundColor: '#f8496033', width: `${(a.quantity / maxAsk) * 100}%` }]} />
                <Text style={styles.askText}>{formatPrice(a.price)}</Text>
                <Text style={styles.qtyText}>{formatQty(a.quantity)}</Text>
              </View>
            ))}
          </View>

          <View style={styles.spreadRow}>
            <Text style={styles.spreadText}>{lastPrice ? formatPrice(lastPrice) : '---'}</Text>
          </View>

          <View style={styles.bidsContainer}>
            {(depth.bids || []).map((b, i) => (
              <View key={`bid-${i}`} style={styles.obRow}>
                <View style={[styles.depthBar, { backgroundColor: '#02c07633', width: `${(b.quantity / maxBid) * 100}%` }]} />
                <Text style={styles.bidText}>{formatPrice(b.price)}</Text>
                <Text style={styles.qtyText}>{formatQty(b.quantity)}</Text>
              </View>
            ))}
          </View>
        </View>
      </ScrollView>

      {/* Action Buttons */}
      <View style={styles.actionContainer}>
        <TouchableOpacity style={[styles.actionBtn, styles.buyBtn]}>
          <Text style={styles.btnText}>AL</Text>
        </TouchableOpacity>
        <TouchableOpacity style={[styles.actionBtn, styles.sellBtn]}>
          <Text style={styles.btnText}>SAT</Text>
        </TouchableOpacity>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#121212' },
  header: { flexDirection: 'row', justifyContent: 'space-between', padding: 16, borderBottomWidth: 1, borderColor: '#2a2a2a' },
  headerTitle: { color: '#fff', fontSize: 18, fontWeight: 'bold' },
  proBadge: { color: '#02c076', fontSize: 12 },
  marketText: { color: '#fff', fontSize: 16, fontFamily: 'monospace' },
  content: { flex: 1, padding: 12 },
  priceContainer: { paddingVertical: 16, alignItems: 'center' },
  bigPrice: { fontSize: 32, fontWeight: 'bold', fontFamily: 'monospace' },
  panel: { backgroundColor: '#1e1e1e', borderRadius: 8, padding: 12, marginBottom: 16 },
  panelTitle: { color: '#848e9c', fontSize: 12, textTransform: 'uppercase', marginBottom: 8 },
  obHeader: { flexDirection: 'row', justifyContent: 'space-between', marginBottom: 4 },
  obHeaderText: { color: '#848e9c', fontSize: 11 },
  obRow: { flexDirection: 'row', justifyContent: 'space-between', paddingVertical: 2, position: 'relative' },
  depthBar: { position: 'absolute', right: 0, top: 0, bottom: 0 },
  askText: { color: '#f84960', fontSize: 13, fontFamily: 'monospace' },
  bidText: { color: '#02c076', fontSize: 13, fontFamily: 'monospace' },
  qtyText: { color: '#fff', fontSize: 13, fontFamily: 'monospace' },
  spreadRow: { paddingVertical: 8, borderTopWidth: 1, borderBottomWidth: 1, borderColor: '#2a2a2a', marginVertical: 4, alignItems: 'center' },
  spreadText: { color: '#02c076', fontSize: 14, fontWeight: 'bold' },
  actionContainer: { flexDirection: 'row', padding: 16, gap: 12, backgroundColor: '#1e1e1e', borderTopWidth: 1, borderColor: '#2a2a2a' },
  actionBtn: { flex: 1, paddingVertical: 14, borderRadius: 8, alignItems: 'center' },
  buyBtn: { backgroundColor: '#02c076' },
  sellBtn: { backgroundColor: '#f84960' },
  btnText: { color: '#fff', fontWeight: 'bold', fontSize: 16 },
});
