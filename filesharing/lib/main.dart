import 'package:filesharing/file_list_screen.dart';
import 'package:filesharing/room_screen.dart';
import 'package:filesharing/theme.dart';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() => runApp(const MyApp());

class MyApp extends StatefulWidget {
  const MyApp({super.key});

  @override
  State<MyApp> createState() => _MyAppState();
}

class _MyAppState extends State<MyApp> {
  ThemeMode _themeMode = ThemeMode.light;
  String? _roomCode;

  @override
  void initState() {
    super.initState();
    _loadPrefs();
  }

  void _loadPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _themeMode =
          (prefs.getBool('isDarkMode') ?? false) ? ThemeMode.dark : ThemeMode.light;
      _roomCode = prefs.getString('roomCode');
    });
  }

  void _toggleTheme() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _themeMode =
          _themeMode == ThemeMode.light ? ThemeMode.dark : ThemeMode.light;
      prefs.setBool('isDarkMode', _themeMode == ThemeMode.dark);
    });
  }

  void _joinRoom(String code) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('roomCode', code);
    setState(() => _roomCode = code);
  }

  void _leaveRoom() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('roomCode');
    setState(() => _roomCode = null);
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      title: 'File Sharing App',
      theme: lightTheme,
      darkTheme: darkTheme,
      themeMode: _themeMode,
      home: _roomCode != null
          ? FileListScreen(
              roomCode: _roomCode!,
              themeMode: _themeMode,
              onThemeChanged: _toggleTheme,
              onLeaveRoom: _leaveRoom,
            )
          : RoomScreen(
              themeMode: _themeMode,
              onThemeChanged: _toggleTheme,
              onRoomJoined: _joinRoom,
            ),
    );
  }
}
