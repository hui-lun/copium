import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import '../../core/format.dart';
import '../../core/providers.dart';
import '../../core/theme.dart';
import '../../models/models.dart';
import '../../shared/widgets.dart';
import 'backtest_report.dart';

/// Configure and run a backtest, then show the report.
class BacktestPage extends ConsumerStatefulWidget {
  const BacktestPage({super.key});

  @override
  ConsumerState<BacktestPage> createState() => _BacktestPageState();
}

class _BacktestPageState extends ConsumerState<BacktestPage> {
  int? _strategyId;
  DateTime _start = DateTime(2021, 1, 1);
  DateTime _end = DateTime.now();
  final _capital = TextEditingController(text: '1000000');
  final _maxPositions = TextEditingController(text: '10');
  final _takeProfit = TextEditingController(text: '30');
  final _stopLoss = TextEditingController(text: '10');
  final _trailing = TextEditingController(text: '0');
  final _maxHold = TextEditingController(text: '60');
  final _feeDiscount = TextEditingController(text: '0.6');

  BacktestStatus? _status;
  bool _starting = false;
  Timer? _poll;

  @override
  void dispose() {
    _poll?.cancel();
    for (final c in [_capital, _maxPositions, _takeProfit, _stopLoss, _trailing, _maxHold, _feeDiscount]) {
      c.dispose();
    }
    super.dispose();
  }

  double _num(TextEditingController c) => double.tryParse(c.text.trim()) ?? 0;

  Future<void> _run(List<Strategy> strategies) async {
    final id = _strategyId ?? strategies.first.id;
    setState(() => _starting = true);
    try {
      final jobId = await ref.read(apiProvider).startBacktest(id, {
        'start': fmtDate(_start),
        'end': fmtDate(_end),
        'initial_capital': _num(_capital),
        'max_positions': _num(_maxPositions).round(),
        'take_profit_pct': _num(_takeProfit),
        'stop_loss_pct': _num(_stopLoss),
        'trailing_stop_pct': _num(_trailing),
        'max_hold_days': _num(_maxHold).round(),
        'fee_discount': _num(_feeDiscount) == 0 ? 1 : _num(_feeDiscount),
      });
      setState(() => _status = BacktestStatus(id: jobId, status: 'running', progress: 0, error: ''));
      _startPolling(jobId);
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message);
    } finally {
      if (mounted) setState(() => _starting = false);
    }
  }

  void _startPolling(int id) {
    _poll?.cancel();
    _poll = Timer.periodic(const Duration(seconds: 1), (timer) async {
      try {
        final s = await ref.read(apiProvider).backtest(id);
        if (!mounted) return timer.cancel();
        setState(() => _status = s);
        if (!s.isRunning) timer.cancel();
      } on ApiException catch (e) {
        timer.cancel();
        if (mounted) showMessage(context, e.message);
      }
    });
  }

  Future<void> _pickDate(bool isStart) async {
    final picked = await showDatePicker(
      context: context,
      initialDate: isStart ? _start : _end,
      firstDate: DateTime(2010),
      lastDate: DateTime.now(),
    );
    if (picked != null) setState(() => isStart ? _start = picked : _end = picked);
  }

  @override
  Widget build(BuildContext context) {
    final status = _status;
    return SafeArea(
      bottom: false,
      child: Column(
        children: [
          const PageHeader(eyebrow: '驗證策略', title: '回測'),
          Expanded(
            child: status?.result != null
                ? BacktestReport(
                    result: status!.result!,
                    onReset: () => setState(() => _status = null),
                  )
                : ListView(
                    children: [
                      if (status != null && status.isRunning) _Progress(progress: status.progress),
                      if (status != null && status.isFailed)
                        SectionCard(
                          child: Text('回測失敗：${status.error}', style: const TextStyle(color: AppColors.up)),
                        ),
                      _form(),
                    ],
                  ),
          ),
        ],
      ),
    );
  }

  Widget _form() {
    final strategies = ref.watch(strategiesProvider);
    final running = _status?.isRunning ?? false;
    return strategies.when(
      loading: () => const Padding(padding: EdgeInsets.all(32), child: Center(child: CircularProgressIndicator())),
      error: (e, _) => MessageView(
        icon: Icons.cloud_off,
        message: e.toString(),
        action: TextButton(onPressed: () => ref.invalidate(strategiesProvider), child: const Text('重試')),
      ),
      data: (list) {
        if (list.isEmpty) return const MessageView(icon: Icons.rule, message: '尚未建立任何策略');
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const _SectionTitle('策略與期間'),
                  DropdownButtonFormField<int>(
                    initialValue: _strategyId ?? list.first.id,
                    decoration: const InputDecoration(labelText: '策略'),
                    items: [for (final s in list) DropdownMenuItem(value: s.id, child: Text(s.name))],
                    onChanged: (v) => setState(() => _strategyId = v),
                  ),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(child: _DateField(label: '開始', date: _start, onTap: () => _pickDate(true))),
                      const SizedBox(width: 12),
                      Expanded(child: _DateField(label: '結束', date: _end, onTap: () => _pickDate(false))),
                    ],
                  ),
                ],
              ),
            ),
            SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const _SectionTitle('資金管理'),
                  _pair(_NumField(label: '總資金', controller: _capital), _NumField(label: '最多同時持有（檔）', controller: _maxPositions)),
                  const SizedBox(height: 8),
                  const Text('每檔投入金額 = 目前資產 ÷ 最多持有檔數，以整張買進。',
                      style: TextStyle(fontSize: 12, color: AppColors.muted)),
                ],
              ),
            ),
            SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const _SectionTitle('出場規則（0 = 不使用）'),
                  _pair(_NumField(label: '停利 %', controller: _takeProfit), _NumField(label: '停損 %', controller: _stopLoss)),
                  const SizedBox(height: 12),
                  _pair(_NumField(label: '移動停利 %', controller: _trailing), _NumField(label: '最長持有（交易日）', controller: _maxHold)),
                ],
              ),
            ),
            SectionCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const _SectionTitle('交易成本'),
                  _NumField(label: '手續費折數（0.6 = 六折）', controller: _feeDiscount),
                  const SizedBox(height: 8),
                  const Text('手續費 0.1425%（最低 20 元）、證交稅 0.3%。訊號隔天以開盤價進場，漲停鎖死買不到、跌停鎖死賣不掉。',
                      style: TextStyle(fontSize: 12, color: AppColors.muted, height: 1.5)),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 24),
              child: FilledButton(
                onPressed: running || _starting ? null : () => _run(list),
                child: Text(running ? '回測進行中…' : '開始回測'),
              ),
            ),
          ],
        );
      },
    );
  }

  Widget _pair(Widget a, Widget b) => Row(children: [Expanded(child: a), const SizedBox(width: 12), Expanded(child: b)]);
}

class _Progress extends StatelessWidget {
  const _Progress({required this.progress});

  final double progress;

  @override
  Widget build(BuildContext context) {
    return SectionCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('回測進行中 ${(progress * 100).toStringAsFixed(0)}%', style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 12),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: LinearProgressIndicator(
              value: progress == 0 ? null : progress,
              minHeight: 8,
              backgroundColor: AppColors.chipBorder,
            ),
          ),
        ],
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle(this.text);

  final String text;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: Text(text, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
      );
}

class _NumField extends StatelessWidget {
  const _NumField({required this.label, required this.controller});

  final String label;
  final TextEditingController controller;

  @override
  Widget build(BuildContext context) => TextField(
        controller: controller,
        keyboardType: const TextInputType.numberWithOptions(decimal: true),
        decoration: InputDecoration(labelText: label),
      );
}

class _DateField extends StatelessWidget {
  const _DateField({required this.label, required this.date, required this.onTap});

  final String label;
  final DateTime date;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: InputDecorator(
          decoration: InputDecoration(labelText: label, suffixIcon: const Icon(Icons.calendar_today, size: 18)),
          child: Text(fmtDate(date)),
        ),
      );
}
