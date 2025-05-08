import '../services/socket_service.dart';
import 'package:flutter/foundation.dart';

class AppStateProvider with ChangeNotifier {

  // Client
  final String clientId;
  bool isReady;
  bool isConnected;

  // UI
  bool isLoading;
  bool showImage;

  // Server
  int readyCount;
  int totalCount;
  String? imageUrl; // Really need?
  String? targetTimeUTC;

  // Functional
  late SocketService socketService;

  AppStateProvider(this.clientId,{
    this.isReady = false,
    this.isConnected = false,
    this.isLoading = false,
    this.showImage = false,
    this.readyCount = 0,
    this.totalCount = 0,
    this.imageUrl,
    this.targetTimeUTC,
  });

  void updateWith({
    String? clientId,
    bool? isReady,
    bool? isConnected,
    bool? isLoading,
    bool? showImage,
    int? readyCount,
    int? totalCount,
    String? overallState,
    String? imageUrl,
    String? targetTimeUTC,
  }) {
    this.isConnected = isConnected ?? this.isConnected;
    this.isLoading = isLoading ?? this.isLoading;
    this.showImage = showImage ?? this.showImage;
    this.readyCount = readyCount ?? this.readyCount;
    this.totalCount = totalCount ?? this.totalCount;
    this.imageUrl = imageUrl ?? this.imageUrl;
    this.targetTimeUTC = targetTimeUTC ?? this.targetTimeUTC;
    
    if (isReady != null) {
      this.isReady = isReady;

      socketService.sendReadyStatus(
        this.clientId,
        isReady,
      );
    }

    notifyListeners();
  }
}