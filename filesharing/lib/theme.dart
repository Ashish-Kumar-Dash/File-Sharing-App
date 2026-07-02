import 'package:flutter/material.dart';

final ThemeData lightTheme = ThemeData(
  useMaterial3: true,
  fontFamily: 'Inter',
  colorScheme: const ColorScheme.light(
    primary: Color(0xFF0065FF),
    onPrimary: Color(0xFFFFFFFF),
    primaryContainer: Color(0xFFD4E3FF),
    onPrimaryContainer: Color(0xFF001D36),
    secondary: Color(0xFF00A8E8),
    onSecondary: Color(0xFFFFFFFF),
    secondaryContainer: Color(0xFFB9E9FF),
    onSecondaryContainer: Color(0xFF001F2A),
    surface: Color(0xFFFDFBFF),
    onSurface: Color(0xFF1B1B1B),
    error: Color(0xFFB00020),
    onError: Color(0xFFFFFFFF),
    outline: Color(0xFF74777F),
  ),
  scaffoldBackgroundColor: const Color(0xFFF7F7F7),
  cardTheme: CardThemeData(
    elevation: 1,
    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
  ),
  inputDecorationTheme: InputDecorationTheme(
    filled: true,
    fillColor: const Color(0xFFF5F5F5),
    contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    enabledBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: const BorderSide(color: Color(0xFFDDDDDD)),
    ),
    focusedBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: const BorderSide(color: Color(0xFF0065FF), width: 2),
    ),
  ),
  visualDensity: VisualDensity.adaptivePlatformDensity,
);

final ThemeData darkTheme = ThemeData(
  useMaterial3: true,
  fontFamily: 'Inter',
  colorScheme: const ColorScheme.dark(
    primary: Color(0xFF3A86FF),
    onPrimary: Color(0xFFFFFFFF),
    primaryContainer: Color(0xFF00458E),
    onPrimaryContainer: Color(0xFFD4E3FF),
    secondary: Color(0xFF00A8E8),
    onSecondary: Color(0xFFFFFFFF),
    secondaryContainer: Color(0xFF004D66),
    onSecondaryContainer: Color(0xFFB9E9FF),
    surface: Color(0xFF1E1E1E),
    onSurface: Color(0xFFE3E2E6),
    error: Color(0xFFD32F2F),
    onError: Color(0xFFFFFFFF),
    outline: Color(0xFF8E9099),
  ),
  scaffoldBackgroundColor: const Color(0xFF121212),
  cardTheme: CardThemeData(
    elevation: 2,
    color: const Color(0xFF1E1E1E),
    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
  ),
  inputDecorationTheme: InputDecorationTheme(
    filled: true,
    fillColor: const Color(0xFF2A2A2A),
    contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    enabledBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: const BorderSide(color: Colors.transparent),
    ),
    focusedBorder: OutlineInputBorder(
      borderRadius: BorderRadius.circular(8),
      borderSide: const BorderSide(color: Color(0xFF3A86FF), width: 2),
    ),
  ),
  visualDensity: VisualDensity.adaptivePlatformDensity,
);
