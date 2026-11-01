import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/models.dart';

/// Override at build time: flutter run --dart-define=API_BASE_URL=http://192.168.1.10:8080
/// The default reaches the host machine from the Android emulator.
const apiBaseUrl = String.fromEnvironment('API_BASE_URL', defaultValue: 'http://10.0.2.2:8080');

/// Development user until real login exists (Phase 5).
const _devUserId = '1';

class ApiException implements Exception {
  ApiException(this.message);
  final String message;

  @override
  String toString() => message;
}

/// Thin wrapper around the Go REST API. Responses are wrapped as {"data": ...}.
class ApiClient {
  ApiClient({Dio? dio})
      : _dio = dio ??
            Dio(BaseOptions(
              baseUrl: '$apiBaseUrl/api/v1',
              connectTimeout: const Duration(seconds: 10),
              receiveTimeout: const Duration(seconds: 60),
              headers: {'X-User-ID': _devUserId},
            ));

  final Dio _dio;

  Future<dynamic> _send(Future<Response<dynamic>> Function() call) async {
    try {
      final body = (await call()).data;
      return body is Map && body.containsKey('data') ? body['data'] : body;
    } on DioException catch (e) {
      final data = e.response?.data;
      if (data is Map && data['error'] != null) throw ApiException(data['error'].toString());
      throw ApiException('無法連線到伺服器（${e.message ?? e.type.name}）');
    }
  }

  Future<dynamic> _get(String path, [Map<String, dynamic>? query]) =>
      _send(() => _dio.get(path, queryParameters: query));

  List<T> _list<T>(Object? data, T Function(Map<String, dynamic>) f) =>
      (data as List? ?? const []).map((e) => f(e as Map<String, dynamic>)).toList();

  // Market
  Future<List<Quote>> quotes() async => _list(await _get('/quotes'), Quote.fromJson);

  Future<List<Candle>> kline(String code, int months) async =>
      _list(await _get('/stocks/$code/kline', {'months': months}), Candle.fromJson);

  // Screener
  Future<List<ScreenHit>> todayScreens() async => _list(await _get('/screens/today'), ScreenHit.fromJson);

  Future<List<Strategy>> strategies() async => _list(await _get('/strategies'), Strategy.fromJson);

  // Backtests
  Future<int> startBacktest(int strategyId, Map<String, dynamic> config) async {
    final data = await _send(() => _dio.post('/backtests', data: {'strategy_id': strategyId, 'config': config}));
    return (data as Map)['id'] as int;
  }

  Future<BacktestStatus> backtest(int id) async =>
      BacktestStatus.fromJson(await _get('/backtests/$id') as Map<String, dynamic>);

  // AI chat: sends the conversation, returns the model's reply.
  Future<String> chat(List<ChatMessage> messages) async {
    final data = await _send(() => _dio.post(
          '/chat',
          data: {
            'messages': [for (final m in messages) if (!m.isError) m.toJson()],
          },
          // Gemini can take well over a minute under load.
          options: Options(receiveTimeout: const Duration(seconds: 100)),
        ));
    return (data as Map)['reply'] as String;
  }

  // Personal data
  Future<List<WatchItem>> watchlist() async => _list(await _get('/watchlist'), WatchItem.fromJson);

  Future<void> addWatch(String code) => _send(() => _dio.post('/watchlist', data: {'code': code}));

  Future<void> removeWatch(String code) => _send(() => _dio.delete('/watchlist/$code'));

  Future<List<Holding>> holdings() async => _list(await _get('/holdings'), Holding.fromJson);

  Future<void> saveHolding(String code, int shares, double avgCost) =>
      _send(() => _dio.put('/holdings/$code', data: {'shares': shares, 'avg_cost': avgCost}));

  Future<void> deleteHolding(String code) => _send(() => _dio.delete('/holdings/$code'));
}

final apiProvider = Provider<ApiClient>((ref) => ApiClient());
