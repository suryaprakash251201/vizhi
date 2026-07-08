import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../services/auth_service.dart';
import '../services/api_service.dart';

final _secureStorageProvider = Provider<FlutterSecureStorage>((ref) {
  return const FlutterSecureStorage();
});

final _dioProvider = Provider<Dio>((ref) {
  return Dio(BaseOptions(
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 30),
  ));
});

final authServiceProvider = Provider<AuthService>((ref) {
  final dio = ref.watch(_dioProvider);
  final storage = ref.watch(_secureStorageProvider);
  return AuthService(dio: dio, storage: storage);
});

final apiServiceProvider = Provider<ApiService>((ref) {
  final dio = ref.watch(_dioProvider);
  return ApiService(dio: dio);
});

class AuthState {
  final bool isAuthenticated;
  final bool isLoading;
  final String? error;
  final String? host;

  const AuthState({
    this.isAuthenticated = false,
    this.isLoading = false,
    this.error,
    this.host,
  });

  AuthState copyWith({
    bool? isAuthenticated,
    bool? isLoading,
    String? error,
    String? host,
  }) {
    return AuthState(
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      host: host ?? this.host,
    );
  }
}

class AuthNotifier extends StateNotifier<AuthState> {
  final AuthService _authService;

  AuthNotifier(this._authService) : super(const AuthState()) {
    _tryRestore();
  }

  Future<void> _tryRestore() async {
    final ok = await _authService.tryRestoreSession();
    if (ok) {
      final host = await _authService.getHost();
      state = AuthState(isAuthenticated: true, host: host);
    }
  }

  Future<void> login(String host, String password) async {
    state = state.copyWith(isLoading: true, error: null);
    try {
      await _authService.login(host, password);
      state = AuthState(isAuthenticated: true, host: host);
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: 'Login failed: ${e.toString()}',
      );
    }
  }

  Future<void> logout() async {
    await _authService.clearCredentials();
    state = const AuthState();
  }

  void clearError() {
    state = state.copyWith(error: null);
  }
}

final authProvider =
    StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  final authService = ref.watch(authServiceProvider);
  return AuthNotifier(authService);
});
