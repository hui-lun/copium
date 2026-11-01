import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../core/format.dart';
import '../../core/theme.dart';
import '../../models/models.dart';

enum KPeriod { day, week, month }

/// Groups daily candles into weekly or monthly candles.
List<Candle> aggregateCandles(List<Candle> daily, KPeriod period) {
  if (period == KPeriod.day || daily.isEmpty) return daily;
  DateTime keyOf(DateTime d) => period == KPeriod.week
      ? DateTime(d.year, d.month, d.day - (d.weekday - 1)) // Monday of that week
      : DateTime(d.year, d.month);

  final out = <Candle>[];
  var bucket = <Candle>[];
  void flush() {
    if (bucket.isEmpty) return;
    out.add(Candle(
      date: bucket.first.date,
      open: bucket.first.open,
      close: bucket.last.close,
      high: bucket.map((c) => c.high).reduce(math.max),
      low: bucket.map((c) => c.low).reduce(math.min),
      volume: bucket.fold(0, (s, c) => s + c.volume),
    ));
    bucket = [];
  }

  for (final c in daily) {
    if (bucket.isNotEmpty && keyOf(bucket.first.date) != keyOf(c.date)) flush();
    bucket.add(c);
  }
  flush();
  return out;
}

/// Candlestick + volume chart with optional buy/sell markers.
class CandleChart extends StatelessWidget {
  const CandleChart({super.key, required this.candles, this.buyDate, this.sellDate});

  final List<Candle> candles;
  final DateTime? buyDate;
  final DateTime? sellDate;

  static const height = 300.0;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: 'K線圖，共 ${candles.length} 根',
      child: LayoutBuilder(
        builder: (context, constraints) => CustomPaint(
          size: Size(constraints.maxWidth, height),
          painter: _CandlePainter(candles, buyDate, sellDate),
        ),
      ),
    );
  }
}

class _CandlePainter extends CustomPainter {
  _CandlePainter(this.candles, this.buyDate, this.sellDate);

  final List<Candle> candles;
  final DateTime? buyDate;
  final DateTime? sellDate;

  static const _slot = 7.0;
  static const _labelWidth = 48.0;
  static const _priceHeight = 200.0;
  static const _volumeTop = 232.0;
  static const _volumeHeight = 56.0;

  @override
  void paint(Canvas canvas, Size size) {
    final count = math.min(candles.length, ((size.width - _labelWidth) / _slot).floor());
    if (count <= 0) return;
    final visible = candles.sublist(candles.length - count);

    var hi = visible.map((c) => c.high).reduce(math.max);
    var lo = visible.map((c) => c.low).reduce(math.min);
    if (hi == lo) {
      hi += 1;
      lo -= 1;
    }
    final pad = (hi - lo) * 0.06;
    hi += pad;
    lo -= pad;
    double y(double v) => (hi - v) / (hi - lo) * _priceHeight;

    final grid = Paint()
      ..color = AppColors.border
      ..strokeWidth = 1;
    for (final gy in [0.0, _priceHeight / 2, _priceHeight]) {
      canvas.drawLine(Offset(0, gy), Offset(size.width, gy), grid);
    }
    _label(canvas, hi.toStringAsFixed(1), Offset(size.width, 2));
    _label(canvas, ((hi + lo) / 2).toStringAsFixed(1), Offset(size.width, _priceHeight / 2 + 2));
    _label(canvas, lo.toStringAsFixed(1), Offset(size.width, _priceHeight - 14));
    _label(canvas, '成交量', const Offset(0, _priceHeight + 10), alignRight: false);

    final maxVol = visible.map((c) => c.volume).reduce(math.max).toDouble();
    final paint = Paint();
    for (var i = 0; i < visible.length; i++) {
      final c = visible[i];
      final x = i * _slot + _slot / 2;
      paint.color = c.close >= c.open ? AppColors.up : AppColors.down;
      canvas.drawRect(Rect.fromLTRB(x - 0.5, y(c.high), x + 0.5, y(c.low)), paint);
      final top = y(math.max(c.open, c.close));
      final bodyH = math.max(1.0, (y(c.open) - y(c.close)).abs());
      canvas.drawRect(Rect.fromLTWH(x - 3, top, 6, bodyH), paint);

      if (maxVol > 0) {
        final vh = c.volume / maxVol * _volumeHeight;
        canvas.drawRect(
          Rect.fromLTWH(x - 3, _volumeTop + _volumeHeight - vh, 6, vh),
          Paint()..color = paint.color.withValues(alpha: 0.7),
        );
      }
    }

    _marker(canvas, visible, buyDate, y, isBuy: true);
    _marker(canvas, visible, sellDate, y, isBuy: false);
  }

  /// Draws a triangle at the candle containing [date]: below for buys, above for sells.
  void _marker(Canvas canvas, List<Candle> visible, DateTime? date, double Function(double) y,
      {required bool isBuy}) {
    if (date == null) return;
    var idx = -1;
    for (var i = 0; i < visible.length; i++) {
      if (!visible[i].date.isAfter(date)) idx = i;
    }
    if (idx < 0) return;
    final c = visible[idx];
    final x = idx * _slot + _slot / 2;
    final color = isBuy ? AppColors.accent : AppColors.info;
    final path = Path();
    if (isBuy) {
      final top = y(c.low) + 4;
      path
        ..moveTo(x, top)
        ..lineTo(x - 5, top + 8)
        ..lineTo(x + 5, top + 8)
        ..close();
      _label(canvas, '買', Offset(x - 6, top + 9), color: color, alignRight: false);
    } else {
      final bottom = y(c.high) - 4;
      path
        ..moveTo(x, bottom)
        ..lineTo(x - 5, bottom - 8)
        ..lineTo(x + 5, bottom - 8)
        ..close();
      _label(canvas, '賣', Offset(x - 6, bottom - 24), color: color, alignRight: false);
    }
    canvas.drawPath(path, Paint()..color = color);
  }

  void _label(Canvas canvas, String text, Offset at, {Color color = AppColors.muted, bool alignRight = true}) {
    final tp = TextPainter(
      text: TextSpan(text: text, style: TextStyle(fontSize: 11, color: color, fontFeatures: tabular)),
      textDirection: TextDirection.ltr,
    )..layout();
    tp.paint(canvas, alignRight ? Offset(at.dx - tp.width, at.dy) : at);
  }

  @override
  bool shouldRepaint(_CandlePainter old) =>
      old.candles != candles || old.buyDate != buyDate || old.sellDate != sellDate;
}

/// "2026-10-30" style label for the latest candle, used under the chart.
String candleRange(List<Candle> candles) =>
    candles.isEmpty ? '' : '${fmtDate(candles.first.date)} ~ ${fmtDate(candles.last.date)}';
