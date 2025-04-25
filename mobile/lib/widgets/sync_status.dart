import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';

class SyncStatus extends StatelessWidget {
  final bool isConnected;
  final int readyCount;
  final int totalCount;
  final bool isReady;
  final VoidCallback onToggleReady;
  final VoidCallback onUploadImage;

  const SyncStatus({
    required this.isConnected,
    required this.readyCount,
    required this.totalCount,
    required this.isReady,
    required this.onToggleReady,
    required this.onUploadImage,
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text('Status: ${isConnected ? 'Connected' : 'Disconnected'}'),
        Text('Ready: $readyCount/$totalCount'),
        ElevatedButton(
          onPressed: onToggleReady,
          child: Text(isReady ? 'Not Ready' : 'I\'m Ready'),
        ),
        if (kDebugMode) ...[
          const SizedBox(height: 20),
          ElevatedButton(
            onPressed: onUploadImage,
            child: const Text('Upload Image (Admin)'),
          ),
        ],
      ],
    );
  }
}