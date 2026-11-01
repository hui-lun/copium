/// Data models matching the Go API's JSON.
library;

double _d(Object? v) => (v as num?)?.toDouble() ?? 0;
int _i(Object? v) => (v as num?)?.toInt() ?? 0;
String _s(Object? v) => v as String? ?? '';
List<T> _list<T>(Object? v, T Function(Map<String, dynamic>) f) =>
    (v as List? ?? const []).map((e) => f(e as Map<String, dynamic>)).toList();

// ---------- market ----------

class Quote {
  const Quote({
    required this.code,
    required this.name,
    required this.exchange,
    required this.date,
    required this.open,
    required this.high,
    required this.low,
    required this.close,
    required this.change,
    required this.volume,
    required this.value,
  });

  final String code;
  final String name;
  final String exchange; // TWSE | TPEx
  final String date;
  final double open, high, low, close, change;
  final int volume; // shares
  final int value; // NT$

  double get prevClose => close - change;
  double get changePct => prevClose > 0 ? change / prevClose * 100 : 0;
  String get exchangeLabel => exchange == 'TPEx' ? '上櫃' : '上市';

  factory Quote.fromJson(Map<String, dynamic> j) => Quote(
        code: _s(j['code']),
        name: _s(j['name']).trim(),
        exchange: _s(j['exchange']),
        date: _s(j['date']).split('T').first,
        open: _d(j['open']),
        high: _d(j['high']),
        low: _d(j['low']),
        close: _d(j['close']),
        change: _d(j['change']),
        volume: _i(j['volume']),
        value: _i(j['value']),
      );
}

class Candle {
  const Candle({
    required this.date,
    required this.open,
    required this.high,
    required this.low,
    required this.close,
    required this.volume,
  });

  final DateTime date;
  final double open, high, low, close;
  final int volume;

  factory Candle.fromJson(Map<String, dynamic> j) => Candle(
        date: DateTime.parse(_s(j['date'])),
        open: _d(j['open']),
        high: _d(j['high']),
        low: _d(j['low']),
        close: _d(j['close']),
        volume: _i(j['volume']),
      );
}

// ---------- screener ----------

class ScreenHit {
  const ScreenHit({
    required this.date,
    required this.strategyId,
    required this.strategyName,
    required this.code,
    required this.name,
    required this.close,
    required this.matched,
  });

  final String date;
  final int strategyId;
  final String strategyName;
  final String code;
  final String name;
  final double close;
  final List<String> matched;

  factory ScreenHit.fromJson(Map<String, dynamic> j) => ScreenHit(
        date: _s(j['date']).split('T').first,
        strategyId: _i(j['strategy_id']),
        strategyName: _s(j['strategy_name']),
        code: _s(j['code']),
        name: _s(j['name']).trim(),
        close: _d(j['close']),
        matched: (j['matched'] as List? ?? const []).map((e) => e.toString()).toList(),
      );
}

class Strategy {
  const Strategy({required this.id, required this.name, required this.enabled});

  final int id;
  final String name;
  final bool enabled;

  factory Strategy.fromJson(Map<String, dynamic> j) =>
      Strategy(id: _i(j['id']), name: _s(j['name']), enabled: j['enabled'] as bool? ?? true);
}

// ---------- backtest ----------

class BacktestStatus {
  const BacktestStatus({
    required this.id,
    required this.status,
    required this.progress,
    required this.error,
    this.result,
  });

  final int id;
  final String status; // running | done | failed
  final double progress; // 0..1
  final String error;
  final BacktestResult? result;

  bool get isRunning => status == 'running';
  bool get isFailed => status == 'failed';

  factory BacktestStatus.fromJson(Map<String, dynamic> j) {
    final b = j['backtest'] as Map<String, dynamic>;
    final r = b['result'];
    return BacktestStatus(
      id: _i(b['id']),
      status: _s(b['status']),
      progress: _d(j['progress']),
      error: _s(b['error']),
      result: r == null ? null : BacktestResult.fromJson(r as Map<String, dynamic>),
    );
  }
}

class BacktestResult {
  const BacktestResult({
    required this.metrics,
    required this.distribution,
    required this.trades,
    required this.equity,
  });

  final Metrics metrics;
  final List<Bucket> distribution;
  final List<Trade> trades;
  final List<EquityPoint> equity;

  factory BacktestResult.fromJson(Map<String, dynamic> j) => BacktestResult(
        metrics: Metrics.fromJson(j['metrics'] as Map<String, dynamic>),
        distribution: _list(j['distribution'], Bucket.fromJson),
        trades: _list(j['trades'], Trade.fromJson),
        equity: _list(j['equity'], EquityPoint.fromJson),
      );
}

class Metrics {
  const Metrics({
    required this.finalEquity,
    required this.totalReturnPct,
    required this.cagrPct,
    required this.maxDrawdownPct,
    required this.benchmarkReturnPct,
    required this.trades,
    required this.stocks,
    required this.winRatePct,
    required this.avgWinPct,
    required this.avgLossPct,
    required this.payoffRatio,
    required this.avgHoldDays,
  });

  final double finalEquity, totalReturnPct, cagrPct, maxDrawdownPct, benchmarkReturnPct;
  final int trades, stocks;
  final double winRatePct, avgWinPct, avgLossPct, payoffRatio, avgHoldDays;

  factory Metrics.fromJson(Map<String, dynamic> j) => Metrics(
        finalEquity: _d(j['final_equity']),
        totalReturnPct: _d(j['total_return_pct']),
        cagrPct: _d(j['cagr_pct']),
        maxDrawdownPct: _d(j['max_drawdown_pct']),
        benchmarkReturnPct: _d(j['benchmark_return_pct']),
        trades: _i(j['trades']),
        stocks: _i(j['stocks']),
        winRatePct: _d(j['win_rate_pct']),
        avgWinPct: _d(j['avg_win_pct']),
        avgLossPct: _d(j['avg_loss_pct']),
        payoffRatio: _d(j['payoff_ratio']),
        avgHoldDays: _d(j['avg_hold_days']),
      );
}

class Trade {
  const Trade({
    required this.code,
    required this.name,
    required this.entryDate,
    required this.exitDate,
    required this.entryPrice,
    required this.exitPrice,
    required this.shares,
    required this.pnl,
    required this.returnPct,
    required this.holdDays,
    required this.exitReason,
  });

  final String code, name, entryDate, exitDate;
  final double entryPrice, exitPrice;
  final int shares;
  final double pnl, returnPct;
  final int holdDays;
  final String exitReason;

  factory Trade.fromJson(Map<String, dynamic> j) => Trade(
        code: _s(j['code']),
        name: _s(j['name']).trim(),
        entryDate: _s(j['entry_date']),
        exitDate: _s(j['exit_date']),
        entryPrice: _d(j['entry_price']),
        exitPrice: _d(j['exit_price']),
        shares: _i(j['shares']),
        pnl: _d(j['pnl']),
        returnPct: _d(j['return_pct']),
        holdDays: _i(j['hold_days']),
        exitReason: _s(j['exit_reason']),
      );
}

class EquityPoint {
  const EquityPoint({required this.date, required this.equity, required this.benchmark});

  final String date;
  final double equity, benchmark;

  factory EquityPoint.fromJson(Map<String, dynamic> j) =>
      EquityPoint(date: _s(j['date']), equity: _d(j['equity']), benchmark: _d(j['benchmark']));
}

class Bucket {
  const Bucket({required this.label, required this.count});

  final String label;
  final int count;

  factory Bucket.fromJson(Map<String, dynamic> j) => Bucket(label: _s(j['label']), count: _i(j['count']));
}

// ---------- personal data ----------

class WatchItem {
  const WatchItem({required this.code});

  final String code;

  factory WatchItem.fromJson(Map<String, dynamic> j) => WatchItem(code: _s(j['code']));
}

class Holding {
  const Holding({required this.code, required this.shares, required this.avgCost});

  final String code;
  final int shares;
  final double avgCost;

  factory Holding.fromJson(Map<String, dynamic> j) =>
      Holding(code: _s(j['code']), shares: _i(j['shares']), avgCost: _d(j['avg_cost']));
}
