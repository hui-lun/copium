import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../core/format.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/widgets.dart';

const _exitReasons = {
  'stop_loss': '停損',
  'trailing_stop': '移動停利',
  'take_profit': '停利',
  'max_hold': '持有到期',
  'signal_lost': '訊號消失',
  'delisted': '下市',
  'open_at_end': '期末未平倉',
};

/// Performance report: metrics, equity curve, return distribution, trades.
class BacktestReport extends StatelessWidget {
  const BacktestReport({super.key, required this.result, required this.onReset});

  final BacktestResult result;
  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final m = result.metrics;
    final trades = result.trades.reversed.take(300).toList(); // newest first
    String pct(double v) => fmtPct(v, withArrow: false);

    return ListView(
      children: [
        SectionCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  const Expanded(child: Text('績效摘要', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700))),
                  TextButton.icon(onPressed: onReset, icon: const Icon(Icons.tune, size: 18), label: const Text('重新設定')),
                ],
              ),
              const SizedBox(height: 8),
              _grid([
                StatTile(label: '總報酬', value: pct(m.totalReturnPct), color: changeColor(m.totalReturnPct)),
                StatTile(label: '0050 同期', value: pct(m.benchmarkReturnPct), color: changeColor(m.benchmarkReturnPct)),
                StatTile(label: '年化報酬', value: pct(m.cagrPct), color: changeColor(m.cagrPct)),
                StatTile(label: '最大回撤', value: pct(m.maxDrawdownPct), color: changeColor(m.maxDrawdownPct)),
                StatTile(label: '勝率', value: '${m.winRatePct.toStringAsFixed(1)}%'),
                StatTile(label: '賺賠比', value: m.payoffRatio.toStringAsFixed(2)),
                StatTile(label: '平均賺', value: pct(m.avgWinPct), color: AppColors.up),
                StatTile(label: '平均賠', value: pct(m.avgLossPct), color: AppColors.down),
                StatTile(label: '交易次數 / 檔數', value: '${m.trades} / ${m.stocks}'),
                StatTile(label: '平均持有', value: '${m.avgHoldDays.toStringAsFixed(1)} 天'),
              ]),
              const SizedBox(height: 12),
              StatTile(label: '期末資產', value: fmtInt(m.finalEquity), size: 22),
            ],
          ),
        ),
        SectionCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('資產曲線', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
              const SizedBox(height: 12),
              _EquityChart(points: result.equity),
              const SizedBox(height: 8),
              const Row(
                children: [
                  _Legend(color: AppColors.accent, label: '策略'),
                  SizedBox(width: 16),
                  _Legend(color: AppColors.muted, label: '0050 買進持有'),
                ],
              ),
            ],
          ),
        ),
        SectionCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text('報酬分布', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
              const SizedBox(height: 4),
              const Text('大海撈針靠少數大賺的股票，看分布比看平均準。',
                  style: TextStyle(fontSize: 12, color: AppColors.muted)),
              const SizedBox(height: 12),
              _Distribution(buckets: result.distribution),
            ],
          ),
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 8, 20, 8),
          child: Text('交易明細（${result.trades.length} 筆）',
              style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
        ),
        for (final t in trades) _TradeRow(trade: t),
        const Disclaimer(),
      ],
    );
  }

  Widget _grid(List<Widget> tiles) => Column(
        children: [
          for (var i = 0; i < tiles.length; i += 2)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Row(
                children: [
                  Expanded(child: tiles[i]),
                  Expanded(child: i + 1 < tiles.length ? tiles[i + 1] : const SizedBox()),
                ],
              ),
            ),
        ],
      );
}

class _Legend extends StatelessWidget {
  const _Legend({required this.color, required this.label});

  final Color color;
  final String label;

  @override
  Widget build(BuildContext context) => Row(
        children: [
          Container(width: 12, height: 3, color: color),
          const SizedBox(width: 6),
          Text(label, style: const TextStyle(fontSize: 12, color: AppColors.muted)),
        ],
      );
}

class _Distribution extends StatelessWidget {
  const _Distribution({required this.buckets});

  final List<Bucket> buckets;

  @override
  Widget build(BuildContext context) {
    final maxCount = buckets.fold<int>(0, (m, b) => math.max(m, b.count));
    return Column(
      children: [
        for (var i = 0; i < buckets.length; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(
              children: [
                SizedBox(width: 110, child: Text(buckets[i].label, style: const TextStyle(fontSize: 13))),
                Expanded(
                  child: Align(
                    alignment: Alignment.centerLeft,
                    child: FractionallySizedBox(
                      widthFactor: maxCount == 0 ? 0 : math.max(0.02, buckets[i].count / maxCount),
                      child: Container(
                        height: 14,
                        decoration: BoxDecoration(
                          // First three buckets are gains (red), the rest losses (green).
                          color: i < 3 ? AppColors.up : AppColors.down,
                          borderRadius: BorderRadius.circular(3),
                        ),
                      ),
                    ),
                  ),
                ),
                SizedBox(
                  width: 44,
                  child: Text('${buckets[i].count}',
                      textAlign: TextAlign.right, style: const TextStyle(fontSize: 13, fontFeatures: tabular)),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

class _TradeRow extends StatelessWidget {
  const _TradeRow({required this.trade});

  final Trade trade;

  @override
  Widget build(BuildContext context) {
    final t = trade;
    final color = changeColor(t.returnPct);
    return InkWell(
      onTap: () => context.push('/stock/${t.code}?buy=${t.entryDate}&sell=${t.exitDate}'),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.divider))),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('${t.name} ${t.code}', style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w500)),
                  const SizedBox(height: 2),
                  Text(
                    '${t.entryDate} → ${t.exitDate} · ${t.holdDays} 天 · ${_exitReasons[t.exitReason] ?? t.exitReason}',
                    style: const TextStyle(fontSize: 12, color: AppColors.muted),
                  ),
                ],
              ),
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Text(fmtPct(t.returnPct),
                    style: TextStyle(fontSize: 15, fontWeight: FontWeight.w500, color: color, fontFeatures: tabular)),
                Text(fmtSigned(t.pnl), style: TextStyle(fontSize: 12, color: color, fontFeatures: tabular)),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _EquityChart extends StatelessWidget {
  const _EquityChart({required this.points});

  final List<EquityPoint> points;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      label: '資產曲線圖',
      child: LayoutBuilder(
        builder: (context, c) => CustomPaint(size: Size(c.maxWidth, 180), painter: _EquityPainter(points)),
      ),
    );
  }
}

class _EquityPainter extends CustomPainter {
  _EquityPainter(this.points);

  final List<EquityPoint> points;

  @override
  void paint(Canvas canvas, Size size) {
    if (points.length < 2) return;
    final values = [
      for (final p in points) ...[p.equity, if (p.benchmark > 0) p.benchmark],
    ];
    var hi = values.reduce(math.max);
    var lo = values.reduce(math.min);
    if (hi == lo) {
      hi += 1;
      lo -= 1;
    }
    double x(int i) => i * size.width / (points.length - 1);
    double y(double v) => (hi - v) / (hi - lo) * (size.height - 16) + 8;

    Path line(double Function(EquityPoint) pick) {
      final path = Path();
      var started = false;
      for (var i = 0; i < points.length; i++) {
        final v = pick(points[i]);
        if (v <= 0) continue;
        started ? path.lineTo(x(i), y(v)) : path.moveTo(x(i), y(v));
        started = true;
      }
      return path;
    }

    final stroke = Paint()
      ..style = PaintingStyle.stroke
      ..strokeJoin = StrokeJoin.round;
    canvas.drawPath(line((p) => p.benchmark), stroke..color = AppColors.muted..strokeWidth = 1.2);
    canvas.drawPath(line((p) => p.equity), stroke..color = AppColors.accent..strokeWidth = 2);

    for (final (v, dy) in [(hi, 0.0), (lo, size.height - 14)]) {
      final tp = TextPainter(
        text: TextSpan(text: fmtInt(v), style: const TextStyle(fontSize: 11, color: AppColors.muted)),
        textDirection: TextDirection.ltr,
      )..layout();
      tp.paint(canvas, Offset(size.width - tp.width, dy));
    }
  }

  @override
  bool shouldRepaint(_EquityPainter old) => old.points != points;
}
