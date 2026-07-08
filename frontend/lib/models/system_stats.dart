class SystemStats {
  final DateTime timestamp;
  final String hostname;
  final int uptimeSeconds;
  final String os;
  final String platform;
  final CPUStats cpu;
  final MemoryStats memory;
  final SwapStats swap;
  final List<DiskStats> disks;
  final NetworkStats network;
  final LoadStats? load;
  final int processCount;
  final List<ProcessInfo> topProcesses;

  SystemStats({
    required this.timestamp,
    required this.hostname,
    required this.uptimeSeconds,
    required this.os,
    required this.platform,
    required this.cpu,
    required this.memory,
    required this.swap,
    required this.disks,
    required this.network,
    this.load,
    required this.processCount,
    required this.topProcesses,
  });

  factory SystemStats.fromJson(Map<String, dynamic> json) {
    return SystemStats(
      timestamp: DateTime.parse(json['timestamp'] as String),
      hostname: json['hostname'] as String? ?? '',
      uptimeSeconds: json['uptime_seconds'] as int? ?? 0,
      os: json['os'] as String? ?? '',
      platform: json['platform'] as String? ?? '',
      cpu: CPUStats.fromJson(json['cpu'] as Map<String, dynamic>),
      memory: MemoryStats.fromJson(json['memory'] as Map<String, dynamic>),
      swap: SwapStats.fromJson(json['swap'] as Map<String, dynamic>),
      disks: (json['disks'] as List<dynamic>?)
              ?.map((e) => DiskStats.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
      network: NetworkStats.fromJson(json['network'] as Map<String, dynamic>),
      load: json['load'] != null
          ? LoadStats.fromJson(json['load'] as Map<String, dynamic>)
          : null,
      processCount: json['process_count'] as int? ?? 0,
      topProcesses: (json['top_processes'] as List<dynamic>?)
              ?.map((e) => ProcessInfo.fromJson(e as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

class CPUStats {
  final double percentUsed;
  final int count;

  CPUStats({required this.percentUsed, required this.count});

  factory CPUStats.fromJson(Map<String, dynamic> json) {
    return CPUStats(
      percentUsed: (json['percent_used'] as num).toDouble(),
      count: json['count'] as int,
    );
  }
}

class MemoryStats {
  final int totalBytes;
  final int availableBytes;
  final int usedBytes;
  final double percent;

  MemoryStats({
    required this.totalBytes,
    required this.availableBytes,
    required this.usedBytes,
    required this.percent,
  });

  factory MemoryStats.fromJson(Map<String, dynamic> json) {
    return MemoryStats(
      totalBytes: json['total_bytes'] as int,
      availableBytes: json['available_bytes'] as int,
      usedBytes: json['used_bytes'] as int,
      percent: (json['percent_used'] as num).toDouble(),
    );
  }
}

class SwapStats {
  final int totalBytes;
  final int usedBytes;
  final double percent;

  SwapStats({required this.totalBytes, required this.usedBytes, required this.percent});

  factory SwapStats.fromJson(Map<String, dynamic> json) {
    return SwapStats(
      totalBytes: json['total_bytes'] as int,
      usedBytes: json['used_bytes'] as int,
      percent: (json['percent_used'] as num).toDouble(),
    );
  }
}

class DiskStats {
  final String mountPoint;
  final String fsType;
  final int totalBytes;
  final int usedBytes;
  final int freeBytes;
  final double percent;

  DiskStats({
    required this.mountPoint,
    required this.fsType,
    required this.totalBytes,
    required this.usedBytes,
    required this.freeBytes,
    required this.percent,
  });

  factory DiskStats.fromJson(Map<String, dynamic> json) {
    return DiskStats(
      mountPoint: json['mount_point'] as String,
      fsType: json['fs_type'] as String? ?? '',
      totalBytes: json['total_bytes'] as int,
      usedBytes: json['used_bytes'] as int,
      freeBytes: json['free_bytes'] as int,
      percent: (json['percent_used'] as num).toDouble(),
    );
  }
}

class NetworkStats {
  final int bytesSent;
  final int bytesRecv;
  final int packetsSent;
  final int packetsRecv;

  NetworkStats({
    required this.bytesSent,
    required this.bytesRecv,
    required this.packetsSent,
    required this.packetsRecv,
  });

  factory NetworkStats.fromJson(Map<String, dynamic> json) {
    return NetworkStats(
      bytesSent: json['bytes_sent'] as int? ?? 0,
      bytesRecv: json['bytes_recv'] as int? ?? 0,
      packetsSent: json['packets_sent'] as int? ?? 0,
      packetsRecv: json['packets_recv'] as int? ?? 0,
    );
  }
}

class LoadStats {
  final double load1;
  final double load5;
  final double load15;

  LoadStats({required this.load1, required this.load5, required this.load15});

  factory LoadStats.fromJson(Map<String, dynamic> json) {
    return LoadStats(
      load1: (json['load_1'] as num).toDouble(),
      load5: (json['load_5'] as num).toDouble(),
      load15: (json['load_15'] as num).toDouble(),
    );
  }
}

class ProcessInfo {
  final int pid;
  final String name;
  final double cpu;
  final double memory;
  final String status;
  final int uptime;

  ProcessInfo({
    required this.pid,
    required this.name,
    required this.cpu,
    required this.memory,
    required this.status,
    required this.uptime,
  });

  factory ProcessInfo.fromJson(Map<String, dynamic> json) {
    return ProcessInfo(
      pid: json['pid'] as int,
      name: json['name'] as String? ?? '',
      cpu: (json['cpu_percent'] as num).toDouble(),
      memory: (json['memory_percent'] as num).toDouble(),
      status: json['status'] as String? ?? 'unknown',
      uptime: json['uptime_seconds'] as int? ?? 0,
    );
  }
}
