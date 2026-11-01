import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/api_client.dart';
import '../../core/format.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/stock_picker.dart';
import '../../shared/widgets.dart';

class WatchlistPage extends ConsumerStatefulWidget {
  const WatchlistPage({super.key});

  @override
  ConsumerState<WatchlistPage> createState() => _WatchlistPageState();
}

class _WatchlistPageState extends ConsumerState<WatchlistPage> {
  // Rows removed by swiping, hidden until the list reloads.
  final _removed = <String>{};

  Future<void> _add() async {
    final code = await pickStock(context, title: '新增自選股');
    if (code == null) return;
    try {
      await ref.read(apiProvider).addWatch(code);
      ref.invalidate(watchlistProvider);
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message);
    }
  }

  Future<void> _remove(String code) async {
    setState(() => _removed.add(code));
    try {
      await ref.read(apiProvider).removeWatch(code);
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message);
    }
    ref.invalidate(watchlistProvider);
  }

  @override
  Widget build(BuildContext context) {
    final items = ref.watch(watchlistProvider);
    final quotes = ref.watch(quoteMapProvider).value ?? const {};
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          PageHeader(
            eyebrow: '我種的韭菜',
            title: '自選股',
            actions: [RoundIconButton(icon: Icons.add, tooltip: '新增自選股', onPressed: _add)],
          ),
          Expanded(
            child: AsyncBody(
              value: items,
              onRetry: () => ref.invalidate(watchlistProvider),
              builder: (list) {
                final visible = list.where((w) => !_removed.contains(w.code)).toList();
                if (visible.isEmpty) {
                  return const MessageView(icon: Icons.star_border, message: '還沒有自選股，按右上角 ＋ 加入。');
                }
                return ListView(
                  children: [
                    for (final w in visible)
                      Dismissible(
                        key: ValueKey(w.code),
                        direction: DismissDirection.endToStart,
                        background: Container(
                          alignment: Alignment.centerRight,
                          padding: const EdgeInsets.only(right: 24),
                          color: AppColors.down.withValues(alpha: 0.2),
                          child: const Icon(Icons.delete_outline, color: AppColors.down),
                        ),
                        onDismissed: (_) => _remove(w.code),
                        child: _WatchRow(code: w.code, quote: quotes[w.code]),
                      ),
                    const Padding(
                      padding: EdgeInsets.all(16),
                      child: Text('向左滑可移除', textAlign: TextAlign.center, style: TextStyle(fontSize: 12, color: AppColors.muted)),
                    ),
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

class _WatchRow extends ConsumerWidget {
  const _WatchRow({required this.code, required this.quote});

  final String code;
  final Quote? quote;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final q = quote;
    final color = changeColor(q?.change ?? 0);
    final closes = ref.watch(klineProvider((code, 1))).value?.map((c) => c.close).toList() ?? const <double>[];
    return InkWell(
      onTap: () => context.push('/stock/$code'),
      child: Container(
        constraints: const BoxConstraints(minHeight: 68),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.divider))),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(q?.name ?? code, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
                  Text(code, style: const TextStyle(fontSize: 12, color: AppColors.muted)),
                ],
              ),
            ),
            Sparkline(values: closes, color: color),
            const SizedBox(width: 12),
            SizedBox(
              width: 92,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Text(
                    fmtPrice(q?.close ?? 0),
                    style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500, color: color, fontFeatures: tabular),
                  ),
                  Text(
                    q == null ? '--' : fmtPct(q.changePct),
                    style: TextStyle(fontSize: 13, color: color, fontFeatures: tabular),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
