class FileInfo {
  final String name;
  final int size;
  final DateTime modTime;

  FileInfo({
    required this.name,
    required this.size,
    required this.modTime,
  });

  factory FileInfo.fromJson(Map<String, dynamic> json) {
    return FileInfo(
      name: json['name'] as String,
      size: json['size'] as int,
      modTime: DateTime.parse(json['mod_time'] as String),
    );
  }
}

class UploadSession {
  final String id;
  final String fileName;
  final int totalSize;
  final int totalChunks;
  final int receivedSize;
  final int receivedChunks;

  UploadSession({
    required this.id,
    required this.fileName,
    required this.totalSize,
    required this.totalChunks,
    required this.receivedSize,
    required this.receivedChunks,
  });

  factory UploadSession.fromJson(Map<String, dynamic> json) {
    return UploadSession(
      id: json['id'] as String,
      fileName: json['file_name'] as String,
      totalSize: json['total_size'] as int,
      totalChunks: json['total_chunks'] as int,
      receivedSize: json['received_size'] as int? ?? 0,
      receivedChunks: json['received_chunks'] as int? ?? 0,
    );
  }

  double get progress => totalChunks > 0 ? receivedChunks / totalChunks : 0;
}
