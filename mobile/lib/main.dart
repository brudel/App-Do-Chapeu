import 'package:flutter/material.dart';
import 'package:multitag/screens/main_screen.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final prefs = await SharedPreferences.getInstance();
  runApp(MultiTagApp(prefs: prefs));
}

class MultiTagApp extends StatelessWidget {
  final SharedPreferences prefs;
  final String serverUrl = '192.168.15.100:8080';

  const MultiTagApp({super.key, required this.prefs});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MultiTag Sync',
      theme: ThemeData(primarySwatch: Colors.blue),
      home: MainScreen(prefs: prefs, serverUrl: serverUrl),
    );
  }
}
