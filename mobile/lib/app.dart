import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'screens/main_screen.dart';

class MultiTagApp extends StatelessWidget {
  final SharedPreferences prefs;
  final String serverUrl;

  const MultiTagApp({
    required this.prefs,
    required this.serverUrl,
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MultiTag Sync',
      theme: ThemeData(primarySwatch: Colors.blue),
      home: MainScreen(prefs: prefs, serverUrl: serverUrl),
    );
  }
}