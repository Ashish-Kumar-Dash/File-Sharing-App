import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

const _baseUrl = String.fromEnvironment(
  'API_BASE_URL',
  defaultValue: 'http://localhost:8000',
);

class RoomScreen extends StatefulWidget {
  final ThemeMode themeMode;
  final VoidCallback onThemeChanged;
  final void Function(String code) onRoomJoined;

  const RoomScreen({
    super.key,
    required this.themeMode,
    required this.onThemeChanged,
    required this.onRoomJoined,
  });

  @override
  State<RoomScreen> createState() => _RoomScreenState();
}

class _RoomScreenState extends State<RoomScreen> {
  final _codeController = TextEditingController();
  bool _isCreating = false;

  @override
  void dispose() {
    _codeController.dispose();
    super.dispose();
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

  Future<void> _createRoom() async {
    setState(() => _isCreating = true);
    try {
      final response = await http.post(Uri.parse('$_baseUrl/rooms'));
      if (!mounted) return;
      if (response.statusCode == 200) {
        final data = jsonDecode(response.body) as Map<String, dynamic>;
        widget.onRoomJoined(data['room'] as String);
      } else {
        _showSnack('Failed to create room', isError: true);
      }
    } catch (e) {
      if (!mounted) return;
      _showSnack('Error: $e', isError: true);
    } finally {
      if (mounted) setState(() => _isCreating = false);
    }
  }

  void _joinRoom() {
    final code = _codeController.text.trim().toUpperCase();
    if (code.length != 6 ||
        !RegExp(r'^[A-HJ-KM-NP-Z2-9]{6}$').hasMatch(code)) {
      _showSnack('Enter a valid 6-character room code', isError: true);
      return;
    }
    widget.onRoomJoined(code);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;

    return Scaffold(
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        toolbarHeight: 180,
        flexibleSpace: ClipRRect(
          borderRadius:
              const BorderRadius.vertical(bottom: Radius.circular(50)),
          child: Container(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                colors: [colorScheme.primary, colorScheme.secondary],
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
              ),
            ),
          ),
        ),
        title: const Text('File Share'),
        titleTextStyle: theme.textTheme.displayMedium?.copyWith(
          color: colorScheme.onPrimary,
        ),
        centerTitle: false,
        actionsIconTheme: IconThemeData(color: colorScheme.onPrimary),
        actions: [
          IconButton(
            icon: Icon(widget.themeMode == ThemeMode.light
                ? Icons.dark_mode_outlined
                : Icons.light_mode_outlined),
            onPressed: widget.onThemeChanged,
            tooltip: 'Toggle Theme',
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 400),
          child: Padding(
            padding: const EdgeInsets.all(24.0),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                SizedBox(
                  width: double.infinity,
                  height: 56,
                  child: FilledButton.icon(
                    onPressed: _isCreating ? null : _createRoom,
                    icon: _isCreating
                        ? const SizedBox(
                            width: 20,
                            height: 20,
                            child:
                                CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.add),
                    label: const Text('Create Room'),
                  ),
                ),
                const SizedBox(height: 32),
                Row(
                  children: [
                    const Expanded(child: Divider()),
                    Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                      child: Text('OR', style: theme.textTheme.bodySmall),
                    ),
                    const Expanded(child: Divider()),
                  ],
                ),
                const SizedBox(height: 32),
                TextField(
                  controller: _codeController,
                  textCapitalization: TextCapitalization.characters,
                  maxLength: 6,
                  decoration: const InputDecoration(
                    labelText: 'Room Code',
                    hintText: 'e.g. A7K2M9',
                    counterText: '',
                  ),
                  onSubmitted: (_) => _joinRoom(),
                ),
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  height: 56,
                  child: OutlinedButton.icon(
                    onPressed: _joinRoom,
                    icon: const Icon(Icons.login),
                    label: const Text('Join Room'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
