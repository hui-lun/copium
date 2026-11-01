import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_client.dart';
import '../../core/format.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/stock_picker.dart';
import '../../shared/widgets.dart';

/// Opens the add/edit holding dialog. Picks a stock first when [code] is null.
Future<void> showHoldingDialog(BuildContext context, WidgetRef ref, {String? code}) async {
  code ??= await pickStock(context, title: '新增持股');
  if (code == null || !context.mounted) return;
  final saved = await showDialog<bool>(context: context, builder: (_) => _HoldingDialog(code: code!));
  if (saved == true) ref.invalidate(holdingsProvider);
}

class PortfolioPage extends ConsumerStatefulWidget {
  const PortfolioPage({super.key});

  @override
  ConsumerState<PortfolioPage> createState() => _PortfolioPageState();
}

class _Row {
  _Row(this.holding, this.quote);

  final Holding holding;
  final Quote? quote;

  /// Falls back to cost when no quote is available.
  double get price => (quote?.close ?? 0) > 0 ? quote!.close : holding.avgCost;
  double get marketValue => holding.shares * price;
  double get cost => holding.shares * holding.avgCost;
  double get pnl => marketValue - cost;
  double get pnlPct => cost > 0 ? pnl / cost * 100 : 0;
}

class _PortfolioPageState extends ConsumerState<PortfolioPage> {
  final _removed = <String>{};

  Future<void> _remove(String code) async {
    setState(() => _removed.add(code));
    try {
      await ref.read(apiProvider).deleteHolding(code);
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message);
    }
    ref.invalidate(holdingsProvider);
  }

  @override
  Widget build(BuildContext context) {
    final holdings = ref.watch(holdingsProvider);
    final quotes = ref.watch(quoteMapProvider).value ?? const {};
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          PageHeader(
            eyebrow: '收成紀錄',
            title: '持倉',
            actions: [
              FilledButton.icon(
                onPressed: () => showHoldingDialog(context, ref),
                icon: const Icon(Icons.add, size: 18),
                label: const Text('新增持股'),
                style: FilledButton.styleFrom(
                  minimumSize: const Size(0, 44),
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  shape: const StadiumBorder(),
                  textStyle: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700),
                ),
              ),
            ],
          ),
          Expanded(
            child: AsyncBody(
              value: holdings,
              onRetry: () => ref.invalidate(holdingsProvider),
              builder: (list) {
                final rows = [
                  for (final h in list)
                    if (!_removed.contains(h.code)) _Row(h, quotes[h.code]),
                ];
                if (rows.isEmpty) {
                  return const MessageView(icon: Icons.work_outline, message: '還沒有持股紀錄，按右上角新增。');
                }
                return ListView(
                  children: [
                    _Summary(rows: rows),
                    const ListHeader(columns: [('名稱 / 股數', null), ('成本 / 現價', 92), ('損益', 100)]),
                    for (var i = 0; i < rows.length; i++)
                      Dismissible(
                        key: ValueKey(rows[i].holding.code),
                        direction: DismissDirection.endToStart,
                        background: Container(
                          alignment: Alignment.centerRight,
                          padding: const EdgeInsets.only(right: 24),
                          color: AppColors.down.withValues(alpha: 0.2),
                          child: const Icon(Icons.delete_outline, color: AppColors.down),
                        ),
                        onDismissed: (_) => _remove(rows[i].holding.code),
                        child: _HoldingRow(row: rows[i], color: AppColors.palette[i % AppColors.palette.length]),
                      ),
                    const Disclaimer(),
                  ],
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _Summary extends StatelessWidget {
  const _Summary({required this.rows});

  final List<_Row> rows;

  @override
  Widget build(BuildContext context) {
    final total = rows.fold<double>(0, (s, r) => s + r.marketValue);
    final cost = rows.fold<double>(0, (s, r) => s + r.cost);
    final pnl = total - cost;
    final ret = cost > 0 ? pnl / cost * 100 : 0.0;
    final trapped = rows.where((r) => r.pnl < 0).length;

    return SectionCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          StatTile(label: '總市值（新台幣）', value: fmtInt(total), size: 32),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(child: StatTile(label: '未實現損益', value: fmtSigned(pnl), color: changeColor(pnl))),
              Expanded(child: StatTile(label: '報酬率', value: fmtPct(ret), color: changeColor(ret))),
            ],
          ),
          const SizedBox(height: 16),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: SizedBox(
              height: 8,
              child: Row(
                children: [
                  for (var i = 0; i < rows.length; i++)
                    Expanded(
                      flex: total > 0 ? (rows[i].marketValue / total * 1000).round().clamp(1, 1000) : 1,
                      child: Container(
                        margin: EdgeInsets.only(right: i == rows.length - 1 ? 0 : 2),
                        color: AppColors.palette[i % AppColors.palette.length],
                      ),
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 6),
          Text('持股配置 · 住套房 $trapped 檔', style: const TextStyle(fontSize: 12, color: AppColors.muted)),
        ],
      ),
    );
  }
}

class _HoldingRow extends StatelessWidget {
  const _HoldingRow({required this.row, required this.color});

  final _Row row;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final h = row.holding;
    final pnlColor = changeColor(row.pnl);
    return InkWell(
      onTap: () => context.push('/stock/${h.code}'),
      child: Container(
        constraints: const BoxConstraints(minHeight: 64),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.divider))),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Container(width: 8, height: 8, decoration: BoxDecoration(color: color, shape: BoxShape.circle)),
                      const SizedBox(width: 8),
                      Flexible(
                        child: Text(row.quote?.name ?? h.code,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
                      ),
                    ],
                  ),
                  Text('${fmtInt(h.shares)} 股',
                      style: const TextStyle(fontSize: 12, color: AppColors.muted, fontFeatures: tabular)),
                ],
              ),
            ),
            SizedBox(
              width: 92,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(fmtPrice(h.avgCost),
                      style: const TextStyle(fontSize: 13, color: AppColors.muted, fontFeatures: tabular)),
                  Text(fmtPrice(row.price), style: const TextStyle(fontSize: 15, fontFeatures: tabular)),
                ],
              ),
            ),
            SizedBox(
              width: 100,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(fmtSigned(row.pnl),
                      style: TextStyle(fontSize: 15, fontWeight: FontWeight.w500, color: pnlColor, fontFeatures: tabular)),
                  Text(fmtPct(row.pnlPct), style: TextStyle(fontSize: 12, color: pnlColor, fontFeatures: tabular)),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _HoldingDialog extends ConsumerStatefulWidget {
  const _HoldingDialog({required this.code});

  final String code;

  @override
  ConsumerState<_HoldingDialog> createState() => _HoldingDialogState();
}

class _HoldingDialogState extends ConsumerState<_HoldingDialog> {
  final _shares = TextEditingController(text: '1000');
  final _cost = TextEditingController();
  bool _saving = false;

  @override
  void dispose() {
    _shares.dispose();
    _cost.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final shares = int.tryParse(_shares.text.trim());
    final cost = double.tryParse(_cost.text.trim());
    if (shares == null || shares < 0 || cost == null || cost <= 0) {
      showMessage(context, '請輸入正確的股數與成本');
      return;
    }
    setState(() => _saving = true);
    try {
      await ref.read(apiProvider).saveHolding(widget.code, shares, cost);
      if (mounted) Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _saving = false);
        showMessage(context, e.message);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final quote = ref.watch(quoteMapProvider).value?[widget.code];
    if (_cost.text.isEmpty && quote != null && quote.close > 0) {
      _cost.text = quote.close.toStringAsFixed(2);
    }
    return AlertDialog(
      title: Text('${quote?.name ?? widget.code}（${widget.code}）'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          TextField(
            controller: _shares,
            keyboardType: TextInputType.number,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: const InputDecoration(labelText: '股數（1 張 = 1000 股）'),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _cost,
            keyboardType: const TextInputType.numberWithOptions(decimal: true),
            decoration: const InputDecoration(labelText: '平均成本'),
          ),
        ],
      ),
      actions: [
        TextButton(onPressed: () => Navigator.of(context).pop(false), child: const Text('取消')),
        TextButton(onPressed: _saving ? null : _save, child: const Text('儲存')),
      ],
    );
  }
}
