import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/app_info.dart';
import '../providers/apps_provider.dart';

class AppsScreen extends ConsumerWidget {
  const AppsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final appsAsync = ref.watch(appsProvider);
    final actionState = ref.watch(appActionProvider);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Row(
            children: [
              Text('Installed Applications',
                  style: Theme.of(context).textTheme.titleMedium),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.refresh),
                onPressed: () => ref.invalidate(appsProvider),
              ),
            ],
          ),
        ),
        if (actionState.error != null)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Text(actionState.error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
          ),
        Expanded(
          child: appsAsync.when(
            data: (apps) => apps.isEmpty
                ? const Center(child: Text('No allowed applications configured'))
                : ListView.builder(
                    itemCount: apps.length,
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    itemBuilder: (context, i) => _AppTile(
                      app: apps[i],
                      isLoading: actionState.isLoading,
                      onLaunch: () =>
                          ref.read(appActionProvider.notifier).launch(apps[i].binary),
                      onTerminate: () =>
                          ref.read(appActionProvider.notifier).terminate(apps[i].binary),
                    ),
                  ),
            error: (err, _) => Center(child: Text('Failed to load apps: $err')),
            loading: () => const Center(child: CircularProgressIndicator()),
          ),
        ),
      ],
    );
  }
}

class _AppTile extends StatelessWidget {
  final AppInfo app;
  final bool isLoading;
  final VoidCallback onLaunch;
  final VoidCallback onTerminate;

  const _AppTile({
    required this.app,
    required this.isLoading,
    required this.onLaunch,
    required this.onTerminate,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: app.running
              ? Colors.green.withOpacity(0.2)
              : Colors.grey.withOpacity(0.2),
          child: Icon(
            app.running ? Icons.check_circle : Icons.circle_outlined,
            color: app.running ? Colors.green : Colors.grey,
          ),
        ),
        title: Text(app.displayName),
        subtitle: Text(app.running
            ? 'Running (PID: ${app.pid ?? "?"})'
            : 'Not running'),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (!app.running)
              FilledButton.tonalIcon(
                onPressed: isLoading ? null : onLaunch,
                icon: const Icon(Icons.play_arrow, size: 18),
                label: const Text('Open'),
              ),
            if (app.running)
              FilledButton.tonalIcon(
                onPressed: isLoading ? null : onTerminate,
                icon: const Icon(Icons.stop, size: 18),
                label: const Text('Close'),
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.red.withOpacity(0.2),
                ),
              ),
          ],
        ),
      ),
    );
  }
}
