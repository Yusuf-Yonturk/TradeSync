import { useEffect, useRef } from 'react';
import { createChart, CandlestickSeries } from 'lightweight-charts';
import './PriceChart.css';

export default function PriceChart({ market, klines, latestTrade }) {
  const chartContainerRef = useRef();
  const chartRef = useRef();
  const candleSeriesRef = useRef();

  // Initialize chart
  useEffect(() => {
    try {
      if (!chartContainerRef.current) return;
      
      const chart = createChart(chartContainerRef.current, {
        autoSize: true,
        layout: {
          background: { type: 'solid', color: '#121212' },
          textColor: '#848e9c',
        },
        grid: {
          vertLines: { color: '#2a2a2a' },
          horzLines: { color: '#2a2a2a' },
        },
        timeScale: {
          timeVisible: true,
          secondsVisible: false,
        },
        crosshair: {
          mode: 0,
        },
        rightPriceScale: {
          autoScale: true,
          scaleMargins: {
            top: 0.1,
            bottom: 0.1,
          },
          mode: 0,
        },
      });

      const series = chart.addSeries(CandlestickSeries, {
        upColor: '#02c076',
        downColor: '#f84960',
        borderVisible: false,
        wickUpColor: '#02c076',
        wickDownColor: '#f84960',
      });
      
      candleSeriesRef.current = series;
      chartRef.current = chart;

      return () => {
        chart.remove();
        chartRef.current = null;
      };
    } catch(err) {
      console.error("Chart init error:", err);
    }
  }, []);

  // Update data when klines change
  useEffect(() => {
    try {
      if (candleSeriesRef.current && klines.length > 0) {
        candleSeriesRef.current.setData(klines);
        if (chartRef.current) {
          chartRef.current.timeScale().fitContent();
        }
      }
    } catch(err) {
      console.error("Chart setData error:", err);
    }
  }, [klines]);

  // Update latest candle with latest trade
  useEffect(() => {
    try {
      if (candleSeriesRef.current && latestTrade && latestTrade.symbol === market) {
        const ts = Math.floor(new Date(latestTrade.traded_at || latestTrade.timestamp).getTime() / 1000);
        const minute = ts - (ts % 60);
        // Normalize price: trades come in raw integer units, divide by 10000
        const tradePrice = latestTrade.price / 10000;
        
        let lastCandle = null;
        if (klines && klines.length > 0) {
          lastCandle = klines[klines.length - 1];
        }

        if (lastCandle && lastCandle.time === minute) {
          // lastCandle values are already normalized (divided by 10000 in api.js)
          candleSeriesRef.current.update({
            time: minute,
            open: lastCandle.open,
            high: Math.max(lastCandle.high, tradePrice),
            low: Math.min(lastCandle.low, tradePrice),
            close: tradePrice
          });
        } else {
          // New candle minute
          candleSeriesRef.current.update({
            time: minute,
            open: tradePrice,
            high: tradePrice,
            low: tradePrice,
            close: tradePrice
          });
        }
      }
    } catch(err) {
      console.error("Chart update error:", err);
    }
  }, [latestTrade, market, klines]);

  return (
    <div className="panel" style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
      <div className="panel-header">{market} / Chart</div>
      <div className="chart-container" ref={chartContainerRef} style={{ flex: 1, minHeight: '400px', width: '100%', position: 'relative' }}>
      </div>
    </div>
  );
}
