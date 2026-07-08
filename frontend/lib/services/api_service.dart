import 'dart:io';
import 'package:dio/dio.dart';
import '../models/system_stats.dart';
import '../models/app_info.dart';
import '../models/file_info.dart';

class ApiService {
  final Dio _dio;

  ApiService({required Dio dio}) : _dio = dio;

  Future<SystemStats> getStats() async {
    final resp = await _dio.get('/api/v1/stats');
    return SystemStats.fromJson(resp.data as Map<String, dynamic>);
  }

  Future<List<AppInfo>> getApps() async {
    final resp = await _dio.get('/api/v1/apps');
    final list = resp.data as List<dynamic>;
    return list
        .map((e) => AppInfo.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<void> launchApp(String binary) async {
    await _dio.post('/api/v1/apps/launch', data: {'binary': binary});
  }

  Future<void> terminateApp(String binary) async {
    await _dio.post('/api/v1/apps/terminate', data: {'binary': binary});
  }

  Future<List<FileInfo>> listFiles() async {
    final resp = await _dio.get('/api/v1/files');
    final list = resp.data as List<dynamic>;
    return list
        .map((e) => FileInfo.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<UploadSession> initUpload(String fileName, int totalSize) async {
    final resp = await _dio.post(
      '/api/v1/files/upload/init',
      data: {'file_name': fileName, 'total_size': totalSize},
    );
    return UploadSession.fromJson(resp.data as Map<String, dynamic>);
  }

  Future<void> uploadChunk(
      String sessionId, int chunkIndex, List<int> data,
      {String? checksumSha}) async {
    final formData = FormData.fromMap({
      'session_id': sessionId,
      'chunk_index': chunkIndex.toString(),
      if (checksumSha != null) 'checksum_sha256': checksumSha,
      'chunk': MultipartFile.fromBytes(data,
          filename: 'chunk_$chunkIndex'),
    });
    await _dio.post('/api/v1/files/upload/chunk', data: formData);
  }

  Future<void> finalizeUpload(String sessionId) async {
    await _dio.post('/api/v1/files/upload/complete',
        data: {'session_id': sessionId});
  }

  Future<File> downloadFile(String path, String savePath) async {
    await _dio.download(
      '/api/v1/files/download/$path',
      savePath,
    );
    return File(savePath);
  }

  Future<void> deleteFile(String path) async {
    await _dio.delete('/api/v1/files/$path');
  }
}
