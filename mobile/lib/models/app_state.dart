class AppState {
  final String clientId;
  final bool isReady;
  final bool isConnected;
  final bool isLoading;
  final bool showImage;
  final int readyCount;
  final int totalCount;
  final String overallState;
  final String? imageUrl;
  final String? targetTimeUTC;

  AppState({
    required this.clientId,
    this.isReady = false,
    this.isConnected = false,
    this.isLoading = false,
    this.showImage = false,
    this.readyCount = 0,
    this.totalCount = 0,
    this.overallState = 'WaitingForUsers',
    this.imageUrl,
    this.targetTimeUTC,
  });

  AppState copyWith({
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
    return AppState(
      clientId: clientId ?? this.clientId,
      isReady: isReady ?? this.isReady,
      isConnected: isConnected ?? this.isConnected,
      isLoading: isLoading ?? this.isLoading,
      showImage: showImage ?? this.showImage,
      readyCount: readyCount ?? this.readyCount,
      totalCount: totalCount ?? this.totalCount,
      overallState: overallState ?? this.overallState,
      imageUrl: imageUrl ?? this.imageUrl,
      targetTimeUTC: targetTimeUTC ?? this.targetTimeUTC,
    );
  }
}