import 'package:copium_app/core/format.dart';
import 'package:copium_app/core/theme.dart';
import 'package:copium_app/features/stock/candle_chart.dart';
import 'package:copium_app/models/models.dart';
import 'package:flutter_test/flutter_test.dart';

Candle _c(String date, double o, double h, double l, double c, int v) =>
    Candle(date: DateTime.parse(date), open: o, high: h, low: l, close: c, volume: v);

void main() {
  test('Taiwan colors: red up, green down', () {
    expect(changeColor(1), AppColors.up);
    expect(changeColor(-1), AppColors.down);
    expect(changeColor(0), AppColors.muted);
  });

  test('percent and number formatting', () {
    expect(fmtPct(1.234), '▲ +1.23%');
    expect(fmtPct(-0.5), '▼ -0.50%');
    expect(fmtPct(2, withArrow: false), '+2.00%');
    expect(fmtLots(32415000), '32,415 張');
    expect(fmtSigned(-1234.4), '-1,234');
    expect(fmtPrice(0), '--');
  });

  test('weekly aggregation merges a Mon-Fri week', () {
    final daily = [
      _c('2026-10-26', 10, 11, 9, 10.5, 100), // Mon
      _c('2026-10-28', 10.5, 13, 10, 12, 200), // Wed
      _c('2026-10-30', 12, 12.5, 8, 9, 300), // Fri
      _c('2026-11-02', 9, 9.5, 8.5, 9.2, 50), // next Mon
    ];
    final weekly = aggregateCandles(daily, KPeriod.week);
    expect(weekly.length, 2);
    final w = weekly.first;
    expect([w.open, w.high, w.low, w.close, w.volume], [10, 13, 8, 9, 600]);
  });

  test('quote change percent uses previous close', () {
    final q = Quote.fromJson({
      'code': '2330', 'name': '台積電', 'exchange': 'TWSE', 'date': '2026-10-30T00:00:00Z',
      'open': 2380, 'high': 2380, 'low': 2360, 'close': 2370, 'change': -45, 'volume': 1, 'value': 1,
    });
    expect(q.prevClose, 2415);
    expect(q.changePct, closeTo(-1.863, 0.001));
    expect(q.date, '2026-10-30');
  });
}
