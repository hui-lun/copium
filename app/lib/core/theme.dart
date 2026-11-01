import 'package:flutter/material.dart';

/// Design tokens from the "韭菜農場 App 畫面" design canvas.
/// Taiwan market convention: red = up, green = down.
abstract final class AppColors {
  static const bg = Color(0xFF0F1115);
  static const surface = Color(0xFF151820);
  static const border = Color(0xFF22262E);
  static const divider = Color(0xFF1E222A);
  static const chipBorder = Color(0xFF2A2F38);
  static const text = Color(0xFFE8EAED);
  static const textSecondary = Color(0xFFC4C8CE);
  static const muted = Color(0xFF9AA0A6);
  static const accent = Color(0xFFF5B544);
  static const onAccent = Color(0xFF1A1405);
  static const up = Color(0xFFFF6B6B);
  static const down = Color(0xFF3DDC84);
  static const info = Color(0xFF6CA8FF);

  /// Categorical colors for allocation charts.
  static const palette = [Color(0xFFF5B544), Color(0xFF6CA8FF), Color(0xFFC792EA), Color(0xFF5FD4C4)];
}

/// Tabular figures keep prices aligned in columns.
const tabular = [FontFeature.tabularFigures()];

ThemeData buildTheme() {
  final base = ThemeData(brightness: Brightness.dark, useMaterial3: true);
  final fieldBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(12),
    borderSide: const BorderSide(color: AppColors.chipBorder),
  );
  return base.copyWith(
    scaffoldBackgroundColor: AppColors.bg,
    colorScheme: const ColorScheme.dark(
      primary: AppColors.accent,
      onPrimary: AppColors.onAccent,
      secondary: AppColors.accent,
      onSecondary: AppColors.onAccent,
      surface: AppColors.surface,
      onSurface: AppColors.text,
      error: AppColors.up,
    ),
    dividerColor: AppColors.divider,
    textTheme: base.textTheme.apply(bodyColor: AppColors.text, displayColor: AppColors.text),
    appBarTheme: const AppBarTheme(
      backgroundColor: AppColors.bg,
      foregroundColor: AppColors.text,
      elevation: 0,
      scrolledUnderElevation: 0,
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: AppColors.surface,
      labelStyle: const TextStyle(color: AppColors.muted),
      hintStyle: const TextStyle(color: AppColors.muted),
      border: fieldBorder,
      enabledBorder: fieldBorder,
      focusedBorder: fieldBorder.copyWith(borderSide: const BorderSide(color: AppColors.accent)),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: AppColors.accent,
        foregroundColor: AppColors.onAccent,
        minimumSize: const Size.fromHeight(48),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: AppColors.accent,
        side: const BorderSide(color: AppColors.accent),
        minimumSize: const Size.fromHeight(48),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
        textStyle: const TextStyle(fontSize: 16, fontWeight: FontWeight.w500),
      ),
    ),
    textButtonTheme: TextButtonThemeData(style: TextButton.styleFrom(foregroundColor: AppColors.accent)),
    snackBarTheme: const SnackBarThemeData(
      backgroundColor: AppColors.surface,
      contentTextStyle: TextStyle(color: AppColors.text),
      behavior: SnackBarBehavior.floating,
    ),
    dialogTheme: const DialogThemeData(backgroundColor: AppColors.surface),
    progressIndicatorTheme: const ProgressIndicatorThemeData(color: AppColors.accent),
  );
}
