import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import '../models/file_info.dart';
import '../providers/upload_provider.dart';
import '../providers/auth_provider.dart';

class FilesScreen extends ConsumerWidget {
  const FilesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final fileList = ref.watch(fileListProvider);
    final upload = ref.watch(uploadProvider);

    return Column(
      children: [
        _buildUploadBar(context, ref, upload),
        Expanded(
          child: fileList.when(
            data: (files) => files.isEmpty
                ? const Center(child: Text('No uploaded files'))
                : ListView.builder(
                    itemCount: files.length,
                    itemBuilder: (context, i) =>
                        _FileTile(file: files[i], ref: ref),
                  ),
            error: (err, _) => Center(child: Text('Error: $err')),
            loading: () => const Center(child: CircularProgressIndicator()),
          ),
        ),
      ],
    );
  }

  Widget _buildUploadBar(
      BuildContext context, WidgetRef ref, UploadState upload) {
    return Container(
      padding: const EdgeInsets.all(16),
      child: Column(
        children: [
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: upload.isUploading
                  ? null
                  : () => ref.read(uploadProvider.notifier).pickAndUpload(),
              icon: upload.isUploading
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.upload_file),
              label: Text(upload.isUploading
                  ? 'Uploading...'
                  : 'Upload File'),
            ),
          ),
          if (upload.isUploading) ...[
            const SizedBox(height: 12),
            LinearProgressIndicator(value: upload.progress),
            const SizedBox(height: 4),
            Text(
              upload.status ?? '',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
          if (upload.error != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(upload.error!,
                  style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                      fontSize: 12)),
            ),
        ],
      ),
    );
  }
}

class _FileTile extends ConsumerWidget {
  final FileInfo file;
  final WidgetRef ref;

  const _FileTile({required this.file, required this.ref});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: ListTile(
        leading: _fileIcon(file.name),
        title: Text(file.name),
        subtitle: Text(_formatSize(file.size)),
        trailing: PopupMenuButton<String>(
          onSelected: (action) {
            if (action == 'delete') {
              _deleteFile(context);
            } else if (action == 'download') {
              _downloadFile(context);
            }
          },
          itemBuilder: (_) => [
            const PopupMenuItem(value: 'download', child: Text('Download')),
            const PopupMenuItem(value: 'delete', child: Text('Delete')),
          ],
        ),
      ),
    );
  }

  Widget _fileIcon(String name) {
    final ext = name.split('.').last.toLowerCase();
    IconData icon;
    switch (ext) {
      case 'txt':
      case 'md':
      case 'log':
        icon = Icons.description;
        break;
      case 'zip':
      case 'tar':
      case 'gz':
        icon = Icons.folder_zip;
        break;
      case 'png':
      case 'jpg':
      case 'jpeg':
      case 'gif':
        icon = Icons.image;
        break;
      case 'pdf':
        icon = Icons.picture_as_pdf;
        break;
      default:
        icon = Icons.insert_drive_file;
    }
    return CircleAvatar(child: Icon(icon));
  }

  void _deleteFile(BuildContext context) async {
    try {
      final api = ref.read(apiServiceProvider);
      await api.deleteFile(file.name);
      ref.invalidate(fileListProvider);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Delete failed: $e')),
        );
      }
    }
  }

  void _downloadFile(BuildContext context) async {
    try {
      final dir = await getApplicationDocumentsDirectory();
      final path = '${dir.path}/${file.name}';
      final api = ref.read(apiServiceProvider);
      await api.downloadFile(file.name, path);
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Saved to $path')),
        );
      }
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Download failed: $e')),
        );
      }
    }
  }

  String _formatSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }
}
