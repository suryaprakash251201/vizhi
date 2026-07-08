import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/system_stats.dart';
import '../services/api_service.dart';
import '../services/ws_service.dart';
import 'auth_provider.dart';

final wsServiceProvider = Provider<WsService>((ref) {
  final ws = WsService();
  ref.onDispose(() => ws.dispose());
  return ws;
});

final statsProvider = StreamProvider<SystemStats>((ref) {
  final ws = ref.watch(wsServiceProvider);
  return ws.statsStream;
});

final wsConnectionProvider = StreamProvider<ConnectionState>((ref) {
  final ws = ref.watch(wsServiceProvider);
  return ws.stateStream;
});

final refreshStatsProvider = FutureProvider<SystemStats>((ref) async {
  final api = ref.watch(apiServiceProvider);
  return api.getStats();
});
