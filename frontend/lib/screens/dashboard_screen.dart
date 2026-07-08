import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/system_stats.dart';
import '../providers/stats_provider.dart';
import '../services/ws_service.dart';
import '../widgets/stats_card.dart';

class DashboardScreen extends ConsumerWidget {
  const DashboardScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final liveStats = ref.watch(statsProvider);
    final wsState = ref.watch(wsConnectionProvider);
    final refresh = ref.watch(refreshStatsProvider);

    return RefreshIndicator(
      onRefresh: () => ref.refresh(refreshStatsProvider.future),
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _buildConnectionBanner(wsState),
          const SizedBox(height: 12),

          liveStats.when(
            data: (stats) => _buildDashboard(context, stats),
            error: (err, _) => refresh.when(
              data: (stats) => _buildDashboard(context, stats),
              error: (e, _) => Center(
                child: Column(
                  children: [
                    Icon(Icons.error_outline, size: 48,
                        color: Theme.of(context).colorScheme.error),
                    const SizedBox(height: 8),
                    Text('Failed to load stats'),
                    TextButton(
                      onPressed: () => ref.refresh(refreshStatsProvider.future),
                      child: const Text('Retry'),
                    ),
                  ],
                ),
              ),
              loading: () => const Center(child: CircularProgressIndicator()),
            ),
            loading: () => const Center(child: CircularProgressIndicator()),
          ),
        ],
      ),
    );
  }

  Widget _buildConnectionBanner(AsyncValue<WsConnectionState> wsState) {
    return wsState.when(
      data: (state) {
        if (state == WsConnectionState.connected) {
          return const Card(
            color: Colors.green,
            child: Padding(
              padding: EdgeInsets.all(8),
              child: Row(
                children: [
                  Icon(Icons.wifi, color: Colors.white, size: 16),
                  SizedBox(width: 8),
                  Text('Live', style: TextStyle(color: Colors.white)),
                ],
              ),
            ),
          );
        }
        return const SizedBox.shrink();
      },
      error: (_, __) => const SizedBox.shrink(),
      loading: () => const SizedBox.shrink(),
    );
  }

  Widget _buildDashboard(BuildContext context, SystemStats stats) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildHeader(context, stats),
        const SizedBox(height: 16),
        _buildMetricGrid(context, stats),
        const SizedBox(height: 16),
        _buildDiskSection(context, stats),
        const SizedBox(height: 16),
        _buildProcessSection(context, stats),
      ],
    );
  }

  Widget _buildHeader(BuildContext context, SystemStats stats) {
    final uptime = Duration(seconds: stats.uptimeSeconds);
    final days = uptime.inDays;
    final hours = uptime.inHours.remainder(24);
    final minutes = uptime.inMinutes.remainder(60);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.computer),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(stats.hostname,
                      style: Theme.of(context).textTheme.titleLarge),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text('${stats.os} — ${stats.platform}'),
            Text('Up $days d $hours h $minutes m  ·  ${stats.processCount} processes'),
          ],
        ),
      ),
    );
  }

  Widget _buildMetricGrid(BuildContext context, SystemStats stats) {
    return Column(
      children: [
        Row(
          children: [
            Expanded(child: StatsCard(
              title: 'CPU',
              value: '${stats.cpu.percentUsed.toStringAsFixed(1)}%',
              icon: Icons.memory,
              color: _percentColor(stats.cpu.percentUsed),
              subtitle: '${stats.cpu.count} cores',
            )),
            const SizedBox(width: 12),
            Expanded(child: StatsCard(
              title: 'RAM',
              value: '${stats.memory.percent.toStringAsFixed(1)}%',
              icon: Icons.storage,
              color: _percentColor(stats.memory.percent),
              subtitle: _formatBytes(stats.memory.usedBytes) +
                  ' / ' +
                  _formatBytes(stats.memory.totalBytes),
            )),
          ],
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(child: StatsCard(
              title: 'Swap',
              value: '${stats.swap.percent.toStringAsFixed(1)}%',
              icon: Icons.swap_vert,
              color: _percentColor(stats.swap.percent),
              subtitle: _formatBytes(stats.swap.usedBytes) +
                  ' / ' +
                  _formatBytes(stats.swap.totalBytes),
            )),
            const SizedBox(width: 12),
            if (stats.load != null)
              Expanded(child: StatsCard(
                title: 'Load',
                value: '${stats.load!.load1.toStringAsFixed(1)}',
                icon: Icons.trending_up,
                color: Theme.of(context).colorScheme.secondary,
                subtitle:
                    '${stats.load!.load5.toStringAsFixed(1)}  ${stats.load!.load15.toStringAsFixed(1)}',
              )),
          ],
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(child: StatsCard(
              title: 'Network ↓',
              value: _formatBytes(stats.network.bytesRecv),
              icon: Icons.arrow_downward,
              color: Theme.of(context).colorScheme.secondary,
            )),
            const SizedBox(width: 12),
            Expanded(child: StatsCard(
              title: 'Network ↑',
              value: _formatBytes(stats.network.bytesSent),
              icon: Icons.arrow_upward,
              color: Theme.of(context).colorScheme.secondary,
            )),
          ],
        ),
      ],
    );
  }

  Widget _buildDiskSection(BuildContext context, SystemStats stats) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Disks', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        ...stats.disks
            .where((d) => d.mountPoint.startsWith('/'))
            .map((d) => Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: StatsCard(
                    title: d.mountPoint,
                    value: '${d.percent.toStringAsFixed(1)}%',
                    icon: Icons.disc_full,
                    color: _percentColor(d.percent),
                    subtitle: '${_formatBytes(d.usedBytes)} / ${_formatBytes(d.totalBytes)}',
                  ),
                )),
      ],
    );
  }

  Widget _buildProcessSection(BuildContext context, SystemStats stats) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Top Processes', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        Card(
          child: Column(
            children: [
              _buildProcessHeader(context),
              ...stats.topProcesses.map((p) => _buildProcessRow(context, p)),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildProcessHeader(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        children: [
          const SizedBox(width: 40, child: Text('PID', style: TextStyle(fontWeight: FontWeight.bold))),
          Expanded(child: Text('Name', style: TextStyle(fontWeight: FontWeight.bold))),
          const SizedBox(width: 60, child: Text('CPU', textAlign: TextAlign.right, style: TextStyle(fontWeight: FontWeight.bold))),
          const SizedBox(width: 60, child: Text('MEM', textAlign: TextAlign.right, style: TextStyle(fontWeight: FontWeight.bold))),
        ],
      ),
    );
  }

  Widget _buildProcessRow(BuildContext context, ProcessInfo proc) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      child: Row(
        children: [
          SizedBox(width: 40, child: Text(proc.pid.toString())),
          Expanded(child: Text(proc.name, overflow: TextOverflow.ellipsis)),
          SizedBox(
            width: 60,
            child: Text(proc.cpu.toStringAsFixed(1),
                textAlign: TextAlign.right),
          ),
          SizedBox(
            width: 60,
            child: Text(proc.memory.toStringAsFixed(1),
                textAlign: TextAlign.right),
          ),
        ],
      ),
    );
  }

  Color _percentColor(double pct) {
    if (pct > 90) return Colors.red;
    if (pct > 70) return Colors.orange;
    if (pct > 50) return Colors.yellow.shade700;
    return Colors.green;
  }

  String _formatBytes(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }
}
