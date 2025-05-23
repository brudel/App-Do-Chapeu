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
    _tryToConnect();
   }

  _tryToConnect() async {

    // print('\t\t\t Creating WebSocket');
    try {
      _channel = WebSocketChannel.connect(Uri.parse('ws://$_serverUrl/ws'));
      _channel.stream.listen(
        _handleServerMessage,
        onError: _onWebSocketError,
        onDone: _onWebSocketDisconnection,
      );
  
      await _channel.ready;
    } on Exception {
      //print('\t\t\t Error creating WebSocket: $e');
      return;
    }

    //print('\t\t WebSocker ready');

    if (_channel.closeCode == null) {
      //print('\t\t Connected successfully');
      _stateProvider.updateWith(isConnected: true);

      _registerClient(_stateProvider.clientId);
    }
  }

  void _onWebSocketError(Object error) {
    //print('\t\t Error on WebSocket: $error');
  }

  void _onWebSocketDisconnection() {
    //print('\t\t Listener disconnected');
    _stateProvider.updateWith(isConnected: false);
    Future.delayed(const Duration(milliseconds: 1000), _tryToConnect);
  }

  void _registerClient(String clientId) {
    _channel.sink.add('{"type":"register","clientId":"$clientId","isReady":${_stateProvider.isReady}}');
  }

  void _handleServerMessage(dynamic message) {
    try {
      final data = jsonDecode(message);
      switch (data['type']) {
        case 'full_state':
          _stateProvider.updateWith(
            readyCount: data['state']['readyCount'] ?? 0,
            totalCount: data['state']['totalCount'] ?? 0,
            imageUrl: 'http://$_serverUrl/image',
          );
          break;

        case 'partial_state':
          _stateProvider.updateWith(
            readyCount: data['readyCount'] ?? _stateProvider.readyCount,
            totalCount: data['totalCount'] ?? _stateProvider.totalCount,
          );
          break;

        case 'image_updated':
          _stateProvider.updateWith(
            imageUrl: 'http://$_serverUrl/image',
          );
          break;
          
        case 'start':
          _stateProvider.updateWith(
            targetTimeUTC: data['targetTimestampUTC'],
          );
          _handleStart();
          break;
      }
    } catch (e) {
      print('Error handling message: $e');
    }
  }

  void sendReadyStatus(String clientId, bool isReady) {
    _channel.sink.add('{"type":"ready","clientId":"$clientId","isReady":$isReady}');
  }

  void dispose() {
    _channel.sink.close();
  }
}