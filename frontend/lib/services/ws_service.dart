import 'dart:async';
import 'dart:convert';
import 'package:web_socket_channel/io.dart';
import '../models/system_stats.dart';

enum WsConnectionState { disconnected, connecting, connected }

class WsService {
  IOWebSocketChannel? _channel;
  WsConnectionState _state = WsConnectionState.disconnected;
  final _statsController = StreamController<SystemStats>.broadcast();
  final _stateController = StreamController<WsConnectionState>.broadcast();
  final _statusController = StreamController<String>.broadcast();
  Timer? _reconnectTimer;
  String _baseUrl = '';
  String _token = '';
  bool _disposed = false;

  Stream<SystemStats> get statsStream => _statsController.stream;
  Stream<WsConnectionState> get stateStream => _stateController.stream;
  Stream<String> get statusStream => _statusController.stream;
  WsConnectionState get state => _state;

  void connect(String baseUrl, String token) {
    _baseUrl = baseUrl;
    _token = token;
    _disposed = false;
    _doConnect();
  }

  void _doConnect() {
    if (_disposed) return;

    final wsUrl = _baseUrl
        .replaceFirst('https://', 'wss://')
        .replaceFirst('http://', 'ws://');
    final uri = Uri.parse('$wsUrl/api/v1/stats/stream');

    try {
      _state = WsConnectionState.connecting;
      _stateController.add(_state);

      _channel = IOWebSocketChannel.connect(
        uri,
        headers: {'Authorization': 'Bearer $_token'},
      );

      _state = WsConnectionState.connected;
      _stateController.add(_state);
      _statusController.add('Connected');

      _channel!.stream.listen(
        (data) {
          try {
            final msg = jsonDecode(data as String) as Map<String, dynamic>;
            if (msg['type'] == 'system_stats') {
              final stats = SystemStats.fromJson(
                  msg['data'] as Map<String, dynamic>);
              _statsController.add(stats);
            }
          } catch (e) {
            _statusController.add('Parse error: $e');
          }
        },
        onError: (error) {
          _statusController.add('WS error: $error');
          _scheduleReconnect();
        },
        onDone: () {
          _statusController.add('Disconnected');
          _scheduleReconnect();
        },
        cancelOnError: false,
      );
    } catch (e) {
      _statusController.add('Connection failed: $e');
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    _state = WsConnectionState.disconnected;
    _stateController.add(_state);
    _channel = null;

    if (_disposed) return;

    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(const Duration(seconds: 5), _doConnect);
  }

  void disconnect() {
    _disposed = true;
    _reconnectTimer?.cancel();
    _channel?.sink.close();
    _channel = null;
    _state = WsConnectionState.disconnected;
    _stateController.add(_state);
  }

  void dispose() {
    disconnect();
    _statsController.close();
    _stateController.close();
    _statusController.close();
  }
}
