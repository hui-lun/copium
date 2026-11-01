import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import '../../core/format.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/widgets.dart';
import '../portfolio/portfolio_page.dart';
import 'candle_chart.dart';

class StockDetailPage extends ConsumerStatefulWidget {
  const StockDetailPage({super.key, required this.code, this.buyDate, this.sellDate});

  final String code;
  final DateTime? buyDate; // backtest entry, marked on the chart
  final DateTime? sellDate; // backtest exit

  @override
  ConsumerState<StockDetailPage> createState() => _StockDetailPageState();
}

class _StockDetailPageState extends ConsumerState<StockDetailPage> {
  KPeriod _period = KPeriod.day;

  /// Months of daily data to load per period.
  static const _months = {KPeriod.day: 6, KPeriod.week: 24, KPeriod.month: 60};

  @override
  void initState() {
    super.initState();
    // Make sure an old backtest trade is inside the loaded window.
    final buy = widget.buyDate;
    if (buy != null && DateTime.now().difference(buy).inDays > 150) _period = KPeriod.week;
  }

  Future<void> _toggleWatch(bool inWatchlist) async {
    final api = ref.read(apiProvider);
    try {
      inWatchlist ? await api.removeWatch(widget.code) : await api.addWatch(widget.code);
      ref.invalidate(watchlistProvider);
      if (mounted) showMessage(context, inWatchlist ? '已從自選移除' : '已加入自選');
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final quote = ref.watch(quoteMapProvider).value?[widget.code];
    final kline = ref.watch(klineProvider((widget.code, _months[_period]!)));
    final inWatchlist = ref.watch(watchlistProvider).value?.any((w) => w.code == widget.code) ?? false;

    return Scaffold(
      appBar: AppBar(
        centerTitle: true,
        title: Column(
          children: [
            Text(quote?.name ?? widget.code, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
            Text(
              quote == null ? widget.code : '${widget.code} · ${quote.exchangeLabel}',
              style: const TextStyle(fontSize: 12, color: AppColors.muted),
            ),
          ],
        ),
        actions: [
          IconButton(
            tooltip: inWatchlist ? '移除自選' : '加入自選',
            icon: Icon(inWatchlist ? Icons.star : Icons.star_border, color: AppColors.accent),
            onPressed: () => _toggleWatch(inWatchlist),
          ),
        ],
      ),
      body: ListView(
        children: [
          if (quote != null) _PriceHeader(quote: quote),
          Container(
            margin: const EdgeInsets.fromLTRB(20, 0, 20, 12),
            padding: const EdgeInsets.all(4),
            decoration: BoxDecoration(color: AppColors.surface, borderRadius: BorderRadius.circular(12)),
            child: Row(
              children: [
                for (final (p, label) in const [(KPeriod.day, '日K'), (KPeriod.week, '週K'), (KPeriod.month, '月K')])
                  Expanded(child: _PeriodButton(label: label, selected: p == _period, onTap: () => setState(() => _period = p))),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20),
            child: SizedBox(
              height: CandleChart.height,
              child: AsyncBody(
                value: kline,
                onRetry: () => ref.invalidate(klineProvider((widget.code, _months[_period]!))),
                builder: (daily) => daily.isEmpty
                    ? const MessageView(icon: Icons.candlestick_chart, message: '尚無歷史資料')
                    : CandleChart(
                        candles: aggregateCandles(daily, _period),
                        buyDate: widget.buyDate,
                        sellDate: widget.sellDate,
                      ),
              ),
            ),
          ),
          const Disclaimer(),
        ],
      ),
      bottomNavigationBar: DecoratedBox(
        decoration: const BoxDecoration(
          color: AppColors.surface,
          border: Border(top: BorderSide(color: AppColors.border)),
        ),
        child: SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(20, 12, 20, 12),
            child: Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => _toggleWatch(inWatchlist),
                    child: Text(inWatchlist ? '移除自選' : '加入自選'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    onPressed: () => showHoldingDialog(context, ref, code: widget.code),
                    child: const Text('記錄持股'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _PriceHeader extends StatelessWidget {
  const _PriceHeader({required this.quote});

  final Quote quote;

  @override
  Widget build(BuildContext context) {
    final color = changeColor(quote.change);
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                fmtPrice(quote.close),
                style: TextStyle(fontSize: 40, fontWeight: FontWeight.w700, color: color, height: 1, fontFeatures: tabular),
              ),
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('${arrow(quote.change)}${quote.change.abs().toStringAsFixed(2)}',
                      style: TextStyle(fontSize: 15, color: color, fontFeatures: tabular)),
                  Text(fmtPct(quote.changePct, withArrow: false),
                      style: TextStyle(fontSize: 15, color: color, fontFeatures: tabular)),
                ],
              ),
            ],
          ),
          const SizedBox(height: 4),
          Text('${quote.date} 收盤', style: const TextStyle(fontSize: 12, color: AppColors.muted)),
          const SizedBox(height: 16),
          GridView.count(
            crossAxisCount: 3,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            childAspectRatio: 2.6,
            children: [
              StatTile(label: '開盤', value: fmtPrice(quote.open), size: 15),
              StatTile(label: '最高', value: fmtPrice(quote.high), size: 15, color: AppColors.up),
              StatTile(label: '最低', value: fmtPrice(quote.low), size: 15, color: AppColors.down),
              StatTile(label: '成交量', value: fmtLots(quote.volume), size: 15),
              StatTile(label: '成交金額', value: fmtYi(quote.value), size: 15),
              StatTile(label: '昨收', value: fmtPrice(quote.prevClose), size: 15),
            ],
          ),
        ],
      ),
    );
  }
}

class _PeriodButton extends StatelessWidget {
  const _PeriodButton({required this.label, required this.selected, required this.onTap});

  final String label;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: selected ? AppColors.chipBorder : Colors.transparent,
      borderRadius: BorderRadius.circular(9),
      child: InkWell(
        borderRadius: BorderRadius.circular(9),
        onTap: onTap,
        child: SizedBox(
          height: 36,
          child: Center(
            child: Text(
              label,
              style: TextStyle(
                fontSize: 14,
                color: selected ? AppColors.accent : AppColors.textSecondary,
                fontWeight: selected ? FontWeight.w700 : FontWeight.w400,
              ),
            ),
          ),
        ),
      ),
    );
  }
}
