import 'dart:convert';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_dropzone/flutter_dropzone.dart';
import 'package:http/http.dart' as http;
import 'package:share_plus/share_plus.dart';
import 'package:url_launcher/url_launcher.dart';

const _baseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8000',
);

class _FileInfo {
  final String name;
  final int size;
  final DateTime uploadedAt;

  _FileInfo(
      {required this.name, required this.size, required this.uploadedAt});

  factory _FileInfo.fromJson(Map<String, dynamic> json) => _FileInfo(
        name: json['name'] as String,
        size: json['size'] as int,
        uploadedAt: DateTime.parse(json['uploadedAt'] as String),
      );
}

class FileListScreen extends StatefulWidget {
  final String roomCode;
  final ThemeMode themeMode;
  final VoidCallback onThemeChanged;
  final VoidCallback onLeaveRoom;

  const FileListScreen({
    super.key,
    required this.roomCode,
    required this.themeMode,
    required this.onThemeChanged,
    required this.onLeaveRoom,
  });

  @override
  State<FileListScreen> createState() => _FileListScreenState();
}

class _FileListScreenState extends State<FileListScreen> {
  List<_FileInfo> _files = [];
  bool _isLoading = false;
  bool _isUploading = false;
  late DropzoneViewController _dropController;

  @override
  void initState() {
    super.initState();
    _refreshFiles();
  }

  void _showSnack(String message, {bool isError = false}) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(message),
        backgroundColor: isError ? Theme.of(context).colorScheme.error : null,
      ),
    );
  }

  Future<void> _refreshFiles() async {
    setState(() => _isLoading = true);
    try {
      final response = await http.get(
        Uri.parse('$_baseUrl/files?room=${widget.roomCode}'),
      );
      if (!mounted) return;
      if (response.statusCode == 200) {
        final List<dynamic> data = jsonDecode(response.body);
        setState(() {
          _files = data
              .map((e) => _FileInfo.fromJson(e as Map<String, dynamic>))
              .toList();
        });
      } else {
        _showSnack('Failed to load files', isError: true);
      }
    } catch (e) {
      if (!mounted) return;
      _showSnack('Error: $e', isError: true);
    } finally {
      if (mounted) setState(() => _isLoading = false);
    }
  }

  Future<void> _uploadBytes(String filename, Uint8List fileBytes) async {
    setState(() => _isUploading = true);
    try {
      var request = http.MultipartRequest(
        'POST',
        Uri.parse('$_baseUrl/upload?room=${widget.roomCode}'),
      );
      request.files.add(
        http.MultipartFile.fromBytes('file', fileBytes, filename: filename),
      );

      var response = await request.send();
      if (!mounted) return;

      if (response.statusCode == 200) {
        _showSnack('$filename uploaded');
        _refreshFiles();
      } else {
        final body = await response.stream.bytesToString();
        _showSnack('Upload failed: $body', isError: true);
      }
    } catch (e) {
      if (!mounted) return;
      _showSnack('Error: $e', isError: true);
    } finally {
      if (mounted) setState(() => _isUploading = false);
    }
  }

  Future<void> _pickAndUpload() async {
    if (_isUploading) return;
    try {
      final result = await FilePicker.platform
          .pickFiles(type: FileType.any, allowMultiple: false);
      if (!mounted || result == null || result.files.isEmpty) return;

      final file = result.files.single;
      Uint8List? bytes;
      if (kIsWeb) {
        bytes = file.bytes;
      } else if (file.path != null) {
        bytes = await file.xFile.readAsBytes();
      }
      if (bytes == null) {
        _showSnack('Could not read file', isError: true);
        return;
      }
      await _uploadBytes(file.name, bytes);
    } catch (e) {
      if (!mounted) return;
      _showSnack('Error: $e', isError: true);
    }
  }

  Future<void> _downloadFile(String name) async {
    try {
      final response = await http.get(
        Uri.parse(
            '$_baseUrl/download?room=${widget.roomCode}&filename=$name'),
      );
      if (!mounted) return;
      if (response.statusCode == 200) {
        await launchUrl(Uri.parse(response.body.trim()));
      } else {
        _showSnack('Download failed', isError: true);
      }
    } catch (e) {
      if (!mounted) return;
      _showSnack('Error: $e', isError: true);
    }
  }

  Future<void> _deleteFile(String name) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete file?'),
        content: Text('Delete "$name"? This cannot be undone.'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          TextButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: const Text('Delete')),
        ],
      ),
    );
    if (confirm != true) return;

    setState(() => _files.removeWhere((f) => f.name == name));

    try {
      final response = await http.delete(
        Uri.parse(
            '$_baseUrl/files?room=${widget.roomCode}&name=$name'),
      );
      if (!mounted) return;
      if (response.statusCode == 204) {
        _showSnack('$name deleted');
      } else {
        _showSnack('Delete failed', isError: true);
        _refreshFiles();
      }
    } catch (e) {
      if (!mounted) return;
      _showSnack('Error: $e', isError: true);
      _refreshFiles();
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

  String _timeAgo(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inDays > 0) return '${diff.inDays}d ago';
    if (diff.inHours > 0) return '${diff.inHours}h ago';
    if (diff.inMinutes > 0) return '${diff.inMinutes}m ago';
    return 'just now';
  }

  IconData _fileIcon(String name) {
    final ext =
        name.contains('.') ? name.split('.').last.toLowerCase() : '';
    switch (ext) {
      case 'jpg':
      case 'jpeg':
      case 'png':
      case 'gif':
        return Icons.image;
      case 'pdf':
        return Icons.picture_as_pdf;
      case 'doc':
      case 'docx':
      case 'txt':
        return Icons.description;
      case 'zip':
        return Icons.folder_zip;
      default:
        return Icons.insert_drive_file;
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return Scaffold(
      appBar: AppBar(
        backgroundColor: colorScheme.primary,
        foregroundColor: colorScheme.onPrimary,
        title: GestureDetector(
          onTap: () async {
            await Clipboard.setData(ClipboardData(text: widget.roomCode));
            if (!mounted) return;
            _showSnack('Room code copied');
          },
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(widget.roomCode),
              const SizedBox(width: 4),
              Icon(Icons.copy,
                  size: 16,
                  color: colorScheme.onPrimary.withAlpha(180)),
            ],
          ),
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _isLoading ? null : _refreshFiles,
            tooltip: 'Refresh',
          ),
          IconButton(
            icon: const Icon(Icons.share),
            onPressed: () => Share.share(
                'Join my file sharing room: ${widget.roomCode}'),
            tooltip: 'Share Room Code',
          ),
          IconButton(
            icon: Icon(widget.themeMode == ThemeMode.light
                ? Icons.dark_mode_outlined
                : Icons.light_mode_outlined),
            onPressed: widget.onThemeChanged,
            tooltip: 'Toggle Theme',
          ),
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: widget.onLeaveRoom,
            tooltip: 'Leave Room',
          ),
        ],
      ),
      body: _buildBody(),
      floatingActionButton: FloatingActionButton(
        onPressed: _isUploading ? null : _pickAndUpload,
        tooltip: 'Upload File',
        child: _isUploading
            ? const SizedBox(
                width: 24,
                height: 24,
                child: CircularProgressIndicator(
                    strokeWidth: 2, color: Colors.white),
              )
            : const Icon(Icons.upload),
      ),
    );
  }

  Widget _buildBody() {
    final content = _isLoading && _files.isEmpty
        ? const Center(child: CircularProgressIndicator())
        : _files.isEmpty
            ? Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.folder_open,
                        size: 64,
                        color: Theme.of(context).colorScheme.outline),
                    const SizedBox(height: 16),
                    const Text('No files shared yet — upload something!'),
                  ],
                ),
              )
            : RefreshIndicator(
                onRefresh: _refreshFiles,
                child: ListView.builder(
                  itemCount: _files.length,
                  itemBuilder: (context, index) {
                    final file = _files[index];
                    return ListTile(
                      leading: Icon(_fileIcon(file.name)),
                      title: Text(file.name,
                          overflow: TextOverflow.ellipsis),
                      subtitle: Text(
                          '${_formatSize(file.size)} · ${_timeAgo(file.uploadedAt)}'),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          IconButton(
                            icon: const Icon(Icons.download),
                            onPressed: () => _downloadFile(file.name),
                            tooltip: 'Download',
                          ),
                          IconButton(
                            icon: const Icon(Icons.delete_outline),
                            onPressed: () => _deleteFile(file.name),
                            tooltip: 'Delete',
                          ),
                        ],
                      ),
                    );
                  },
                ),
              );

    if (!kIsWeb) return content;

    return Stack(
      children: [
        DropzoneView(
          operation: DragOperation.copy,
          cursor: CursorType.grab,
          onCreated: (ctrl) => _dropController = ctrl,
          onDropFile: (file) async {
            if (_isUploading) {
              _showSnack('Upload already in progress', isError: true);
              return;
            }
            final bytes = await _dropController.getFileData(file);
            await _uploadBytes(file.name, bytes);
          },
        ),
        content,
      ],
    );
  }
}
