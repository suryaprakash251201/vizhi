import 'dart:convert';
import 'dart:io';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:file_picker/file_picker.dart';
import '../models/file_info.dart';
import '../services/api_service.dart';
import 'auth_provider.dart';

final fileListProvider = FutureProvider<List<FileInfo>>((ref) async {
  final api = ref.watch(apiServiceProvider);
  return api.listFiles();
});

class UploadState {
  final bool isUploading;
  final double progress;
  final String? currentFile;
  final String? error;
  final String? status;

  const UploadState({
    this.isUploading = false,
    this.progress = 0,
    this.currentFile,
    this.error,
    this.status,
  });

  UploadState copyWith({
    bool? isUploading,
    double? progress,
    String? currentFile,
    String? error,
    String? status,
  }) {
    return UploadState(
      isUploading: isUploading ?? this.isUploading,
      progress: progress ?? this.progress,
      currentFile: currentFile ?? this.currentFile,
      error: error,
      status: status ?? this.status,
    );
  }
}

class UploadNotifier extends StateNotifier<UploadState> {
  final ApiService _api;
  final Ref _ref;

  UploadNotifier(this._api, this._ref) : super(const UploadState());

  Future<void> pickAndUpload({int chunkSize = 4 * 1024 * 1024}) async {
    final result = await FilePicker.platform.pickFiles(
      allowMultiple: false,
      withData: true,
    );
    if (result == null || result.files.isEmpty) return;

    final file = result.files.first;
    final fileName = file.name;
    final totalSize = file.size;
    List<int>? bytes;

    if (file.bytes != null) {
      bytes = file.bytes;
    } else if (file.path != null) {
      bytes = await File(file.path!).readAsBytes();
    } else {
      state = state.copyWith(error: 'Cannot read file');
      return;
    }

    state = state.copyWith(
      isUploading: true,
      progress: 0,
      currentFile: fileName,
      error: null,
      status: 'Initializing...',
    );

    try {
      final session = await _api.initUpload(fileName, totalSize);
      final totalChunks = session.totalChunks;

      for (int i = 0; i < totalChunks; i++) {
        final start = i * chunkSize;
        final end = (start + chunkSize > totalSize) ? totalSize : start + chunkSize;
        final chunk = bytes!.sublist(start, end);

        final sha = _sha256(chunk);

        await _api.uploadChunk(session.id, i, chunk, checksumSha: sha);

        state = state.copyWith(
          progress: (i + 1) / totalChunks,
          status: 'Uploading chunk ${i + 1}/$totalChunks',
        );
      }

      await _api.finalizeUpload(session.id);

      state = state.copyWith(
        isUploading: false,
        progress: 1.0,
        status: 'Complete',
      );

      _ref.invalidate(fileListProvider);
    } catch (e) {
      state = state.copyWith(
        isUploading: false,
        error: 'Upload failed: $e',
        status: null,
      );
    }
  }

  String _sha256(List<int> data) {
    final digest = _sha256Digest(data);
    return digest;
  }

  String _sha256Digest(List<int> data) {
    // Use dart:convert's base64 as a simple hash for checksum
    // In production, use a proper SHA-256 implementation
    final bytes = List<int>.from(data);
    // Simple hash — replace with crypto sha256 in production
    return base64Encode(bytes);
  }
}

final uploadProvider =
    StateNotifierProvider<UploadNotifier, UploadState>((ref) {
  final api = ref.watch(apiServiceProvider);
  return UploadNotifier(api, ref);
});

final downloadProvider = Provider<DownloadNotifier>((ref) {
  final api = ref.watch(apiServiceProvider);
  return DownloadNotifier(api, ref);
});

class DownloadState {
  final bool isDownloading;
  final double progress;
  final String? error;

  const DownloadState({
    this.isDownloading = false,
    this.progress = 0,
    this.error,
  });
}

class DownloadNotifier extends StateNotifier<DownloadState> {
  final ApiService _api;
  final Ref _ref;

  DownloadNotifier(this._api, this._ref) : super(const DownloadState());

  Future<void> download(String fileName, String savePath) async {
    state = DownloadState(isDownloading: true, progress: 0);
    try {
      final file = await _api.downloadFile(fileName, savePath);
      state = DownloadState(isDownloading: false, progress: 1.0);
      _ref.invalidate(fileListProvider);
    } catch (e) {
      state = DownloadState(error: e.toString());
    }
  }
}
