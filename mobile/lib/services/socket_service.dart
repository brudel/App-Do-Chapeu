import 'dart:convert';
import 'package:multitag/models/app_state.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

class SocketService {
  late WebSocketChannel _channel;
  final String _serverUrl;
  final AppState _appState;
  final void Function() _handleStart;

  SocketService({
    required String serverUrl,
    required AppState appState,
    required void Function() handleStart,
  }) :
      _serverUrl = serverUrl,
      _appState = appState,
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
  _registerClient(_appState.clientId);
    
    setState(() => _appState = _appState.copyWith(isConnected: true));
  }
  
  void _handleServerMessage(dynamic message) {
    print('Received: $message');
    try {
      final data = jsonDecode(message);
      switch (data['type']) {
        case 'full_state':
          setState(() => _appState = _appState.copyWith(
            readyCount: data['state']['readyCount'] ?? 0,
            totalCount: data['state']['totalCount'] ?? 0,
            overallState: data['state']['overallState'] ?? 'WaitingForUsers',
            imageUrl: data['state']['hasImage'] == true
              ? '${_serverUrl.replaceFirst('ws:', 'http:')}/image'
              : null,
          ));
          break;
          
        case 'partial_state':
          setState(() => _appState = _appState.copyWith(
            readyCount: data['readyCount'] ?? 0,
            totalCount: data['totalCount'] ?? 0,
          ));
          break;
          
        case 'image_updated':
          setState(() => _appState = _appState.copyWith(
            imageUrl: 'http://$_serverUrl/image?t=${DateTime.now().millisecondsSinceEpoch}',
          ));
          break;
          
        case 'start':
          setState(() => _appState = _appState.copyWith(
            targetTimeUTC: data['targetTimestampUTC'],
          ));
          _handleStart();
          break;
      }
    } catch (e) {
      print('Error handling message: $e');
    }
  }

  void _handleDisconnection() {
    setState(() => _appState = _appState.copyWith(isConnected: false));
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