import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';
import 'package:vibration/vibration.dart';
import '../models/app_state_provider.dart';
import '../services/socket_service.dart';
import '../services/image_service.dart';
import '../widgets/sync_status.dart';
import '../widgets/image_display.dart';

class MainScreen extends StatelessWidget {
  final SharedPreferences prefs;
  final String serverUrl;

  const MainScreen({
    required this.prefs,
    required this.serverUrl,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: createAppState,

      child: Consumer<AppStateProvider>(
        builder: layoutBuilder,
      ),
    );
  }

  Widget layoutBuilder(context, provider, child) {
    return Scaffold(
      appBar: AppBar(title: const Text('MultiTag Sync')),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: contentBuilder(provider, context),
        ),
      ),
    );
  }

  List<Widget> contentBuilder(provider, context) {
    return [
      if (provider.isLoading)
        const CircularProgressIndicator()
      else if (provider.showImage && provider.imageUrl != null)
        ImageDisplay(imageUrl: serverUrl)
      else
        SyncStatus(
          isConnected: provider.isConnected,
          readyCount: provider.readyCount,
          totalCount: provider.totalCount,
          isReady: provider.isReady,
          onToggleReady: () => provider.updateWith(isReady: !provider.isReady),
          onUploadImage: () => ImageService.uploadImage(
            serverUrl,
            context,
          ),
        ),
    ];
  }

  AppStateProvider createAppState(context) {
      String? clientId = prefs.getString('clientId');
      if (clientId == null) {
        clientId = const Uuid().v4();
        prefs.setString('clientId', clientId);
      }

      final appStateProvider = AppStateProvider(clientId);

      appStateProvider.socketService = SocketService( // Changed _socketService to socketService
        serverUrl: serverUrl,
        stateProvider: appStateProvider, // This will be the new parameter name
        handleStart: () => _handleStart(appStateProvider),
      );

      return appStateProvider;
    }

  Future<void> _handleStart(AppStateProvider provider) async {
    if (provider.targetTimeUTC == null) return;

    final targetTime = DateTime.parse(provider.targetTimeUTC!);
    final now = DateTime.now().toUtc();

    // 1. Vibrate
    if (await Vibration.hasVibrator()) {
      Vibration.vibrate(duration: 500);
    }

    provider.updateWith(isLoading: true);

    // 2. Show loading screen for 3 seconds
    final delay = targetTime.difference(now);

    // 3. Show image
    if (!delay.isNegative) {
      await Future.delayed(delay);
    }
    provider.updateWith(
      isLoading: false,
      showImage: true,
    );
  }
}