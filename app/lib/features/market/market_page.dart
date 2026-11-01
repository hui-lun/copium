import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../core/format.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/widgets.dart';

enum _Segment { twse, tpex, hot }

/// Market quotes list. TODO: TAIEX / TPEx index cards from the design need an index endpoint.
class MarketPage extends ConsumerStatefulWidget {
  const MarketPage({super.key});

  @override
  ConsumerState<MarketPage> createState() => _MarketPageState();
}

class _MarketPageState extends ConsumerState<MarketPage> {
  _Segment _segment = _Segment.twse;
  bool _searching = false;
  String _query = '';

  List<Quote> _filter(List<Quote> all) {
    final q = _query.trim();
    Iterable<Quote> list = all.where((s) => s.close > 0);
    if (q.isNotEmpty) list = list.where((s) => s.code.startsWith(q) || s.name.contains(q));
    switch (_segment) {
      case _Segment.twse:
        return list.where((s) => s.exchange == 'TWSE').toList();
      case _Segment.tpex:
        return list.where((s) => s.exchange == 'TPEx').toList();
      case _Segment.hot: // top turnover across both markets
        final sorted = list.toList()..sort((a, b) => b.value.compareTo(a.value));
        return sorted.take(100).toList();
    }
  }

  @override
  Widget build(BuildContext context) {
    final quotes = ref.watch(allQuotesProvider);
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          PageHeader(
            eyebrow: '韭菜農場',
            title: '行情',
            actions: [
              RoundIconButton(
                icon: _searching ? Icons.close : Icons.search,
                tooltip: _searching ? '關閉搜尋' : '搜尋股票',
                onPressed: () => setState(() {
                  _searching = !_searching;
                  _query = '';
                }),
              ),
            ],
          ),
          if (_searching)
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
              child: TextField(
                autofocus: true,
                decoration: const InputDecoration(hintText: '輸入代號或名稱', prefixIcon: Icon(Icons.search)),
                onChanged: (v) => setState(() => _query = v),
              ),
            ),
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: PillTabs<_Segment>(
              options: const [(_Segment.twse, '上市'), (_Segment.tpex, '上櫃'), (_Segment.hot, '熱門')],
              selected: _segment,
              onChanged: (v) => setState(() => _segment = v),
            ),
          ),
          const ListHeader(columns: [('名稱 / 代號', null), ('成交價', 96), ('漲跌幅', 88)]),
          Expanded(
            child: AsyncBody(
              value: quotes,
              onRetry: () => ref.invalidate(allQuotesProvider),
              builder: (all) {
                final list = _filter(all);
                return RefreshIndicator(
                  onRefresh: () async {
                    ref.invalidate(allQuotesProvider);
                    await ref.read(allQuotesProvider.future);
                  },
                  child: ListView.builder(
                    physics: const AlwaysScrollableScrollPhysics(),
                    itemCount: list.length,
                    itemBuilder: (context, i) => QuoteRow(quote: list[i]),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class QuoteRow extends StatelessWidget {
  const QuoteRow({super.key, required this.quote});

  final Quote quote;

  @override
  Widget build(BuildContext context) {
    final color = changeColor(quote.change);
    return InkWell(
      onTap: () => context.push('/stock/${quote.code}'),
      child: Container(
        constraints: const BoxConstraints(minHeight: 60),
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
        decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.divider))),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(quote.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500)),
                  Text(quote.code, style: const TextStyle(fontSize: 12, color: AppColors.muted)),
                ],
              ),
            ),
            SizedBox(
              width: 96,
              child: Text(
                fmtPrice(quote.close),
                textAlign: TextAlign.right,
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w500, color: color, fontFeatures: tabular),
              ),
            ),
            SizedBox(
              width: 88,
              child: Text(
                fmtPct(quote.changePct),
                textAlign: TextAlign.right,
                style: TextStyle(fontSize: 14, color: color, fontFeatures: tabular),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
