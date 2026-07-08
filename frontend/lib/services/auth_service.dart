import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class AuthService {
  final Dio _dio;
  final FlutterSecureStorage _storage;
  static const _tokenKey = 'vizhi_token';
  static const _hostKey = 'vizhi_host';

  AuthService({required Dio dio, required FlutterSecureStorage storage})
      : _dio = dio,
        _storage = storage;

  Future<String?> getToken() => _storage.read(key: _tokenKey);
  Future<String?> getHost() => _storage.read(key: _hostKey);

  Future<void> saveCredentials(String host, String token) async {
    await _storage.write(key: _hostKey, value: host);
    await _storage.write(key: _tokenKey, value: token);
    _dio.options.baseUrl = host;
    _dio.options.headers['Authorization'] = 'Bearer $token';
  }

  Future<void> clearCredentials() async {
    await _storage.deleteAll();
    _dio.options.headers.remove('Authorization');
  }

  Future<String> login(String host, String password) async {
    final dio = Dio(BaseOptions(
      baseUrl: host,
      connectTimeout: const Duration(seconds: 10),
      receiveTimeout: const Duration(seconds: 10),
    ));

    final resp = await dio.post(
      '/auth/login',
      data: {'password': password},
    );

    final token = resp.data['token'] as String;
    await saveCredentials(host, token);
    return token;
  }

  Future<bool> tryRestoreSession() async {
    final token = await getToken();
    final host = await getHost();
    if (token == null || host == null) return false;

    _dio.options.baseUrl = host;
    _dio.options.headers['Authorization'] = 'Bearer $token';

    try {
      final resp = await _dio.get('/health');
      return resp.statusCode == 200;
    } catch (_) {
      await clearCredentials();
      return false;
    }
  }
}
