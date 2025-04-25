import 'dart:convert';
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
    Key? key,
  }) : super(key: key);

  @override
  State<MainScreen> createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  late AppState _state;
  late SocketService _socketService;

  @override
  void initState() {
    super.initState();
    _state = AppState(
      clientId: widget.prefs.getString('clientId') ?? const Uuid().v4(),
    );
    widget.prefs.setString('clientId', _state.clientId);
    _initSocket();
  }

  void _initSocket() {
    _socketService = SocketService(
      serverUrl: widget.serverUrl,
      onMessage: _handleServerMessage,
      onDisconnect: _handleDisconnection,
    );
    _socketService.registerClient(_state.clientId);
    setState(() => _state = _state.copyWith(isConnected: true));
  }

  void _handleServerMessage(dynamic message) {
    print('Received: $message');
    try {
      final data = jsonDecode(message);
      switch (data['type']) {
        case 'full_state':
          setState(() => _state = _state.copyWith(
            readyCount: data['state']['readyCount'] ?? 0,
            totalCount: data['state']['totalCount'] ?? 0,
            overallState: data['state']['overallState'] ?? 'WaitingForUsers',
            imageUrl: data['state']['hasImage'] == true
              ? '${widget.serverUrl.replaceFirst('ws:', 'http:')}/image'
              : null,
          ));
          break;
          
        case 'partial_state':
          setState(() => _state = _state.copyWith(
            readyCount: data['readyCount'] ?? 0,
            totalCount: data['totalCount'] ?? 0,
          ));
          break;
          
        case 'image_updated':
          setState(() => _state = _state.copyWith(
            imageUrl: '${widget.serverUrl.replaceFirst('ws:', 'http:')}/image?t=${DateTime.now().millisecondsSinceEpoch}',
          ));
          break;
          
        case 'start':
          setState(() => _state = _state.copyWith(
            targetTimeUTC: data['targetTimestampUTC'],
          ));
          _scheduleSyncActions();
          break;
      }
    } catch (e) {
      print('Error handling message: $e');
    }
  }

  void _handleDisconnection() {
    setState(() => _state = _state.copyWith(isConnected: false));
    Future.delayed(const Duration(seconds: 5), () {
      if (!mounted) return;
      _initSocket();
    });
  }

  void _scheduleSyncActions() async {
    if (_state.targetTimeUTC == null) return;
    
    final targetTime = DateTime.parse(_state.targetTimeUTC!);
    final now = DateTime.now().toUtc();
    final delay = targetTime.difference(now);
    
    if (delay.isNegative) return;
    
    await Future.delayed(delay);
    
    // 1. Vibrate
    if (await Vibration.hasVibrator()) {
      Vibration.vibrate(duration: 500);
    }
    
    // 2. Show loading screen for 3 seconds
    setState(() => _state = _state.copyWith(isLoading: true));
    await Future.delayed(const Duration(seconds: 3));
    setState(() => _state = _state.copyWith(isLoading: false));
    
    // 3. Show image
    setState(() => _state = _state.copyWith(showImage: true));
  }

  void _toggleReady() {
    setState(() => _state = _state.copyWith(isReady: !_state.isReady));
    _socketService.sendReadyStatus(_state.clientId, _state.isReady);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('MultiTag Sync')),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            if (_state.isLoading)
              const CircularProgressIndicator()
            else if (_state.showImage && _state.imageUrl != null)
              ImageDisplay(imageUrl: _state.imageUrl!)
            else
              SyncStatus(
                isConnected: _state.isConnected,
                readyCount: _state.readyCount,
                totalCount: _state.totalCount,
                isReady: _state.isReady,
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