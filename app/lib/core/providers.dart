import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/models.dart';
import 'api_client.dart';

/// All TWSE + TPEx quotes; shared by the market, watchlist and portfolio tabs.
final allQuotesProvider = FutureProvider<List<Quote>>((ref) => ref.watch(apiProvider).quotes());

/// Quotes keyed by code.
final quoteMapProvider = FutureProvider<Map<String, Quote>>((ref) async {
  final quotes = await ref.watch(allQuotesProvider.future);
  return {for (final q in quotes) q.code: q};
});

/// Daily candles for (code, months).
final klineProvider = FutureProvider.autoDispose.family<List<Candle>, (String, int)>(
  (ref, arg) => ref.watch(apiProvider).kline(arg.$1, arg.$2),
);

final screensProvider = FutureProvider.autoDispose<List<ScreenHit>>((ref) => ref.watch(apiProvider).todayScreens());

final strategiesProvider = FutureProvider.autoDispose<List<Strategy>>((ref) => ref.watch(apiProvider).strategies());

final watchlistProvider = FutureProvider.autoDispose<List<WatchItem>>((ref) => ref.watch(apiProvider).watchlist());

final holdingsProvider = FutureProvider.autoDispose<List<Holding>>((ref) => ref.watch(apiProvider).holdings());
