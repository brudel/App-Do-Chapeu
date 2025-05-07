import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';
import 'package:vibration/vibration.dart';
import '../models/app_state.dart';
import '../services/socket_service.dart';
import '../services/image_service.dart';
import '../widgets/sync_status.dart';
import '../widgets/image_display.dart';

class MainScreen extends StatefulWidget {
  final SharedPreferences prefs;
  final String serverUrl;

  const MainScreen({
    required this.prefs,
    required this.serverUrl,
    super.key,
  });

  @override
  State<MainScreen> createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  late AppState _appState;
  late SocketService _socketService;

  @override
  void initState() {
    super.initState();
    String? clientId = widget.prefs.getString('clientId');

    if (clientId ==  null) {
      clientId = const Uuid().v4();
      widget.prefs.setString('clientId', clientId);
    }

    _appState = AppState(clientId: clientId);

    _socketService = SocketService(
      serverUrl: widget.serverUrl,
      appState: _appState,
      handleStart: handleStart,
    );
  }

  void _toggleReady() {
    setState(() => _appState = _appState.copyWith(isReady: !_appState.isReady));
    _socketService.sendReadyStatus(_appState.clientId, _appState.isReady);
  }

  void handleStart() async {
    if (_appState.targetTimeUTC == null) return;

    final targetTime = DateTime.parse(_appState.targetTimeUTC!);
    final now = DateTime.now().toUtc();
    final delay = targetTime.difference(now);
    
    if (delay.isNegative) return;
    
    await Future.delayed(delay);
    
    // 1. Vibrate
    if (await Vibration.hasVibrator()) {
      Vibration.vibrate(duration: 500);
    }
    
    // 2. Show loading screen for 3 seconds
    setState(() => _appState = _appState.copyWith(isLoading: true));
    await Future.delayed(const Duration(seconds: 3));
    setState(() => _appState = _appState.copyWith(isLoading: false));

    // 3. Show image
    setState(() => _appState = _appState.copyWith(showImage: true));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('MultiTag Sync')),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            if (_appState.isLoading)
              const CircularProgressIndicator()
            else if (_appState.showImage && _appState.imageUrl != null)
              ImageDisplay(imageUrl: _appState.imageUrl!)
            else
              SyncStatus(
                isConnected: _appState.isConnected,
                readyCount: _appState.readyCount,
                totalCount: _appState.totalCount,
                isReady: _appState.isReady,
                onToggleReady: _toggleReady,
                onUploadImage: () => ImageService.uploadImage(
                  widget.serverUrl,
                  context,
                ),
              ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _socketService.dispose();
    super.dispose();
  }
}