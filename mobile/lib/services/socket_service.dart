import 'dart:convert';
import 'package:multitag/models/app_state_provider.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

class SocketService {
  late WebSocketChannel _channel;
  final String _serverUrl;
  final AppStateProvider _stateProvider;
  final void Function() _handleStart;

  SocketService({
    required String serverUrl,
    required AppStateProvider stateProvider, // Changed from AppState to AppStateProvider
    required void Function() handleStart,
  }) :
      _serverUrl = serverUrl,
      _stateProvider = stateProvider, // Changed from _appState to _stateProvider
      _handleStart = handleStart
   {
    _initSocket(serverUrl);
   }

  _initSocket(String serverUrl) {
    _channel = WebSocketChannel.connect(Uri.parse('ws://$_serverUrl/ws'));
    _channel.stream.listen(
      _handleServerMessage,
      onDone: _handleDisconnection,
      onError: (error) => _handleDisconnection(),
    );
    _registerClient(_stateProvider.clientId); // Access clientId via provider
    
    _stateProvider.updateWith(isConnected: true); // Use provider to update state
  }
  
  void _handleServerMessage(dynamic message) {
    print('Received: $message');
    try {
      final data = jsonDecode(message);
      switch (data['type']) {
        case 'full_state':
          _stateProvider.updateWith( // Use provider to update state
            readyCount: data['state']['readyCount'] ?? 0,
            totalCount: data['state']['totalCount'] ?? 0,
            imageUrl: data['state']['hasImage'] == true
              ? '${_serverUrl.replaceFirst('ws:', 'http:')}/image'
              : null,
          );
          break;
          
        case 'partial_state':
          _stateProvider.updateWith( // Use provider to update state
            readyCount: data['readyCount'] ?? _stateProvider.readyCount,
            totalCount: data['totalCount'] ?? _stateProvider.totalCount,
          );
          break;
          
        case 'image_updated':
          _stateProvider.updateWith( // Use provider to update state
            imageUrl: 'http://$_serverUrl/image',
          );
          break;
          
        case 'start':
          _stateProvider.updateWith( // Use provider to update state
            targetTimeUTC: data['targetTimestampUTC'],
          );
          _handleStart();
          break;
      }
    } catch (e) {
      print('Error handling message: $e');
    }
  }

  void _handleDisconnection() {
    _stateProvider.updateWith(isConnected: false); // Use provider to update state
    Future.delayed(const Duration(seconds: 5), () {
      _initSocket(_serverUrl);
    });
  }

  void _registerClient(String clientId) {
    _channel.sink.add('{"type":"register","clientId":"$clientId"}');
  }

  void sendReadyStatus(String clientId, bool isReady) {
    _channel.sink.add('{"type":"ready","clientId":"$clientId","isReady":$isReady}');
  }

  void dispose() {
    _channel.sink.close();
  }
}