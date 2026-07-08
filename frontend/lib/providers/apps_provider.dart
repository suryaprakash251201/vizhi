import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/app_info.dart';
import '../services/api_service.dart';
import 'auth_provider.dart';

final appsProvider = FutureProvider<List<AppInfo>>((ref) async {
  final api = ref.watch(apiServiceProvider);
  return api.getApps();
});

class AppActionState {
  final bool isLoading;
  final String? error;
  final String? action;

  const AppActionState({this.isLoading = false, this.error, this.action});
}

class AppActionNotifier extends StateNotifier<AppActionState> {
  final ApiService _api;
  final Ref _ref;

  AppActionNotifier(this._api, this._ref) : super(const AppActionState());

  Future<void> launch(String binary) async {
    state = AppActionState(isLoading: true, action: 'launch');
    try {
      await _api.launchApp(binary);
      state = const AppActionState(action: 'launch');
      _ref.invalidate(appsProvider);
    } catch (e) {
      state = AppActionState(error: e.toString());
    }
  }

  Future<void> terminate(String binary) async {
    state = AppActionState(isLoading: true, action: 'terminate');
    try {
      await _api.terminateApp(binary);
      state = const AppActionState(action: 'terminate');
      _ref.invalidate(appsProvider);
    } catch (e) {
      state = AppActionState(error: e.toString());
    }
  }
}

final appActionProvider =
    StateNotifierProvider<AppActionNotifier, AppActionState>((ref) {
  final api = ref.watch(apiServiceProvider);
  return AppActionNotifier(api, ref);
});
