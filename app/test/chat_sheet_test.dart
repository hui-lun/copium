import 'package:copium_app/core/theme.dart';
import 'package:copium_app/features/chat/chat_sheet.dart';
import 'package:copium_app/models/models.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

/// Pre-filled history, as if the sheet had been used and closed before.
class _SeededChat extends ChatNotifier {
  @override
  List<ChatMessage> build() => [
        for (var i = 1; i <= 30; i++) ...[
          ChatMessage(role: 'user', text: 'question $i'),
          ChatMessage(role: 'model', text: 'answer $i'),
        ],
      ];
}

void main() {
  testWidgets('reopening the chat shows the newest messages, not the oldest', (tester) async {
    await tester.pumpWidget(ProviderScope(
      overrides: [chatProvider.overrideWith(_SeededChat.new)],
      child: MaterialApp(theme: buildTheme(), home: const Scaffold(body: ChatSheet())),
    ));
    await tester.pumpAndSettle();

    expect(find.text('answer 30'), findsOneWidget);
    expect(find.text('question 1'), findsNothing);
  });
}
