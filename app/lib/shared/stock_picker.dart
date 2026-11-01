import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/format.dart';
import '../core/providers.dart';
import '../core/theme.dart';

/// Dialog that searches all quotes and returns the chosen stock code.
Future<String?> pickStock(BuildContext context, {String title = '選擇股票'}) {
  return showDialog<String>(context: context, builder: (_) => _StockPickerDialog(title: title));
}

class _StockPickerDialog extends ConsumerStatefulWidget {
  const _StockPickerDialog({required this.title});

  final String title;

  @override
  ConsumerState<_StockPickerDialog> createState() => _StockPickerDialogState();
}

class _StockPickerDialogState extends ConsumerState<_StockPickerDialog> {
  String _query = '';

  @override
  Widget build(BuildContext context) {
    final quotes = ref.watch(allQuotesProvider).value ?? const [];
    final q = _query.trim();
    final matches = q.isEmpty
        ? const []
        : quotes.where((s) => s.code.startsWith(q) || s.name.contains(q)).take(30).toList();

    return AlertDialog(
      title: Text(widget.title),
      contentPadding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      content: SizedBox(
        width: 360,
        height: 420,
        child: Column(
          children: [
            TextField(
              autofocus: true,
              decoration: const InputDecoration(hintText: '輸入代號或名稱', prefixIcon: Icon(Icons.search)),
              onChanged: (v) => setState(() => _query = v),
            ),
            const SizedBox(height: 8),
            Expanded(
              child: ListView.builder(
                itemCount: matches.length,
                itemBuilder: (context, i) {
                  final s = matches[i];
                  return ListTile(
                    title: Text(s.name),
                    subtitle: Text('${s.code} · ${s.exchangeLabel}', style: const TextStyle(color: AppColors.muted)),
                    trailing: Text(
                      fmtPrice(s.close),
                      style: TextStyle(color: changeColor(s.change), fontFeatures: tabular),
                    ),
                    onTap: () => Navigator.of(context).pop(s.code),
                  );
                },
              ),
            ),
          ],
        ),
      ),
      actions: [TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('取消'))],
    );
  }
}
