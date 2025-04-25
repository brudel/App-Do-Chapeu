import 'package:web_socket_channel/web_socket_channel.dart';

class SocketService {
  final WebSocketChannel channel;
  final Function(dynamic) onMessage;
  final void Function() onDisconnect;

  SocketService({
    required String serverUrl,
    required this.onMessage,
    required this.onDisconnect,
  }) : channel = WebSocketChannel.connect(Uri.parse(serverUrl)) {
    channel.stream.listen(
      onMessage,
      onDone: onDisconnect,
      onError: (error) => onDisconnect(),
    );
  }

  void registerClient(String clientId) {
    channel.sink.add('{"type":"register","clientId":"$clientId"}');
  }

  void sendReadyStatus(String clientId, bool isReady) {
    channel.sink.add('{"type":"ready","clientId":"$clientId","isReady":$isReady}');
  }

  void dispose() {
    channel.sink.close();
  }
}