import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/format.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/widgets.dart';
import '../chat/chat_sheet.dart';

/// "Today's needles": stocks matched by the daily screens.
class ScreenerPage extends ConsumerStatefulWidget {
  const ScreenerPage({super.key});

  @override
  ConsumerState<ScreenerPage> createState() => _ScreenerPageState();
}

class _ScreenerPageState extends ConsumerState<ScreenerPage> {
  int? _strategyId; // null = all strategies

  Future<void> _refresh() async {
    ref.invalidate(screensProvider);
    await ref.read(screensProvider.future);
  }

  @override
  Widget build(BuildContext context) {
    final hits = ref.watch(screensProvider);
    return Scaffold(
      floatingActionButton: FloatingActionButton(
        tooltip: '問 AI',
        onPressed: () => showChatSheet(context),
        backgroundColor: AppColors.accent,
        foregroundColor: AppColors.onAccent,
        child: const Icon(Icons.chat_bubble_outline),
      ),
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            const PageHeader(eyebrow: '韭菜農場', title: '今日撈到的針'),
            Expanded(
              child: AsyncBody(
                value: hits,
                onRetry: () => ref.invalidate(screensProvider),
                builder: (all) => RefreshIndicator(onRefresh: _refresh, child: _list(all)),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _list(List<ScreenHit> all) {
    final strategies = <int, String>{for (final h in all) h.strategyId: h.strategyName};
    final shown = _strategyId == null ? all : all.where((h) => h.strategyId == _strategyId).toList();

    return ListView(
      physics: const AlwaysScrollableScrollPhysics(),
      children: [
        if (all.isEmpty)
          const MessageView(
            icon: Icons.search_off,
            message: '目前沒有撈到符合條件的股票。\n每個交易日晚上會自動執行選股。',
          )
        else ...[
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
            child: Text(
              '${all.first.date} 收盤 · 共 ${shown.length} 檔',
              style: const TextStyle(fontSize: 13, color: AppColors.muted),
            ),
          ),
          if (strategies.length > 1)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: PillTabs<int?>(
                options: [(null, '全部'), for (final e in strategies.entries) (e.key, e.value)],
                selected: _strategyId,
                onChanged: (v) => setState(() => _strategyId = v),
              ),
            ),
          for (final h in shown) _HitTile(hit: h, showStrategy: strategies.length > 1),
        ],
        const Disclaimer(),
        const SizedBox(height: 72), // keep the last row clear of the chat button
      ],
    );
  }
}

class _HitTile extends StatelessWidget {
  const _HitTile({required this.hit, required this.showStrategy});

  final ScreenHit hit;
  final bool showStrategy;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () => context.push('/stock/${hit.code}'),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
        decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.divider))),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(hit.name, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
                  const SizedBox(height: 2),
                  Text(
                    showStrategy ? '${hit.code} · ${hit.strategyName}' : hit.code,
                    style: const TextStyle(fontSize: 12, color: AppColors.muted),
                  ),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 6,
                    runSpacing: 6,
                    children: [for (final m in hit.matched) _ConditionChip(label: m)],
                  ),
                ],
              ),
            ),
            Text(
              fmtPrice(hit.close),
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500, fontFeatures: tabular),
            ),
          ],
        ),
      ),
    );
  }
}

class _ConditionChip extends StatelessWidget {
  const _ConditionChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.accent.withValues(alpha: 0.6)),
      ),
      child: Text(label, style: const TextStyle(fontSize: 12, color: AppColors.accent)),
    );
  }
}
