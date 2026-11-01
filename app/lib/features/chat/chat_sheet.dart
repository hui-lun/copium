import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../core/api_client.dart';
import '../../core/theme.dart';
import '../../models/models.dart';

/// Conversation history, kept for the app session so it survives closing the sheet.
class ChatNotifier extends Notifier<List<ChatMessage>> {
  @override
  List<ChatMessage> build() => const [];

  void add(ChatMessage m) => state = [...state, m];

  void clear() => state = const [];
}

final chatProvider = NotifierProvider<ChatNotifier, List<ChatMessage>>(ChatNotifier.new);

/// Opens the AI chat as a tall bottom sheet.
Future<void> showChatSheet(BuildContext context) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: AppColors.bg,
    shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
    builder: (_) => const FractionallySizedBox(heightFactor: 0.9, child: ChatSheet()),
  );
}

class ChatSheet extends ConsumerStatefulWidget {
  const ChatSheet({super.key});

  @override
  ConsumerState<ChatSheet> createState() => _ChatSheetState();
}

class _ChatSheetState extends ConsumerState<ChatSheet> {
  final _input = TextEditingController();
  final _scroll = ScrollController();
  bool _sending = false;

  static const _suggestions = ['今天撈到哪些股票？', '解釋一下這些選股條件', '這些股票可能有什麼風險？'];

  @override
  void dispose() {
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _send(String text) async {
    text = text.trim();
    if (text.isEmpty || _sending) return;
    final chat = ref.read(chatProvider.notifier);
    chat.add(ChatMessage(role: 'user', text: text));
    _input.clear();
    setState(() => _sending = true);
    _scrollToEnd();
    try {
      final reply = await ref.read(apiProvider).chat(ref.read(chatProvider));
      chat.add(ChatMessage(role: 'model', text: reply));
    } on ApiException catch (e) {
      chat.add(ChatMessage(role: 'error', text: e.message));
    } finally {
      if (mounted) setState(() => _sending = false);
      _scrollToEnd();
    }
  }

  void _scrollToEnd() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(_scroll.position.maxScrollExtent,
            duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final messages = ref.watch(chatProvider);
    return Padding(
      // Keep the input above the on-screen keyboard.
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: Column(
        children: [
          _header(messages.isNotEmpty),
          const Divider(height: 1, color: AppColors.border),
          Expanded(
            child: messages.isEmpty
                ? _welcome()
                : ListView.builder(
                    controller: _scroll,
                    padding: const EdgeInsets.all(16),
                    itemCount: messages.length + (_sending ? 1 : 0),
                    itemBuilder: (context, i) =>
                        i == messages.length ? const _TypingBubble() : _Bubble(message: messages[i]),
                  ),
          ),
          _inputBar(),
        ],
      ),
    );
  }

  Widget _header(bool hasMessages) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 12, 8, 12),
      child: Row(
        children: [
          const Icon(Icons.auto_awesome, color: AppColors.accent, size: 22),
          const SizedBox(width: 8),
          const Expanded(
            child: Text('AI 韭菜顧問', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
          ),
          if (hasMessages)
            IconButton(
              tooltip: '清除對話',
              icon: const Icon(Icons.delete_sweep_outlined, color: AppColors.muted),
              onPressed: _sending ? null : () => ref.read(chatProvider.notifier).clear(),
            ),
          IconButton(
            tooltip: '關閉',
            icon: const Icon(Icons.close, color: AppColors.muted),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }

  Widget _welcome() {
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        const Text(
          '嗨！我可以幫你解讀今天撈到的股票、解釋選股條件和回測結果。',
          style: TextStyle(fontSize: 15, height: 1.6),
        ),
        const SizedBox(height: 20),
        const Text('試試看：', style: TextStyle(fontSize: 13, color: AppColors.muted)),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final s in _suggestions)
              ActionChip(
                label: Text(s),
                onPressed: () => _send(s),
                backgroundColor: AppColors.surface,
                side: const BorderSide(color: AppColors.chipBorder),
                labelStyle: const TextStyle(color: AppColors.textSecondary),
              ),
          ],
        ),
      ],
    );
  }

  Widget _inputBar() {
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 8, 8, 8),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(top: BorderSide(color: AppColors.border)),
      ),
      child: SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Expanded(
                  child: TextField(
                    controller: _input,
                    minLines: 1,
                    maxLines: 4,
                    textInputAction: TextInputAction.send,
                    onSubmitted: _send,
                    decoration: const InputDecoration(
                      hintText: '問問 AI…',
                      isDense: true,
                      fillColor: AppColors.bg,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton.filled(
                  tooltip: '送出',
                  onPressed: _sending ? null : () => _send(_input.text),
                  icon: const Icon(Icons.send),
                  style: IconButton.styleFrom(
                    fixedSize: const Size(44, 44),
                    backgroundColor: AppColors.accent,
                    foregroundColor: AppColors.onAccent,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 6),
            const Text(
              'AI 回答可能有誤，僅供參考，非投資建議。',
              style: TextStyle(fontSize: 11, color: AppColors.muted),
            ),
          ],
        ),
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.message});

  final ChatMessage message;

  @override
  Widget build(BuildContext context) {
    final user = message.isUser;
    final Color bg = user ? AppColors.accent : AppColors.surface;
    final Color fg = user ? AppColors.onAccent : (message.isError ? AppColors.up : AppColors.text);
    return Align(
      alignment: user ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 10),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        constraints: BoxConstraints(maxWidth: MediaQuery.sizeOf(context).width * 0.8),
        decoration: BoxDecoration(
          color: bg,
          border: user ? null : Border.all(color: message.isError ? AppColors.up : AppColors.border),
          borderRadius: BorderRadius.only(
            topLeft: const Radius.circular(16),
            topRight: const Radius.circular(16),
            bottomLeft: Radius.circular(user ? 16 : 4),
            bottomRight: Radius.circular(user ? 4 : 16),
          ),
        ),
        child: SelectableText(
          _plain(message.text),
          style: TextStyle(color: fg, fontSize: 15, height: 1.5),
        ),
      ),
    );
  }

  /// Gemini answers in Markdown; strip the most common markers for plain display.
  static String _plain(String s) => s
      .replaceAllMapped(RegExp(r'\*\*(.+?)\*\*'), (m) => m[1]!)
      .replaceAllMapped(RegExp(r'^#{1,6}\s*', multiLine: true), (_) => '')
      .replaceAllMapped(RegExp(r'^\s*[\*\-]\s+', multiLine: true), (_) => '• ');
}

class _TypingBubble extends StatelessWidget {
  const _TypingBubble();

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 10),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        decoration: BoxDecoration(
          color: AppColors.surface,
          border: Border.all(color: AppColors.border),
          borderRadius: BorderRadius.circular(16),
        ),
        child: const Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2)),
            SizedBox(width: 10),
            Text('思考中…', style: TextStyle(color: AppColors.muted)),
          ],
        ),
      ),
    );
  }
}
