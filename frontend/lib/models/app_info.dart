class AppInfo {
  final String name;
  final String binary;
  final String displayName;
  final bool running;
  final int? pid;

  AppInfo({
    required this.name,
    required this.binary,
    required this.displayName,
    required this.running,
    this.pid,
  });

  factory AppInfo.fromJson(Map<String, dynamic> json) {
    return AppInfo(
      name: json['name'] as String,
      binary: json['binary'] as String,
      displayName: json['display_name'] as String? ?? json['name'] as String,
      running: json['running'] as bool? ?? false,
      pid: json['pid'] as int?,
    );
  }
}
