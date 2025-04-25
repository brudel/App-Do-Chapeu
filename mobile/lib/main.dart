import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:uuid/uuid.dart';
import 'package:http/http.dart' as http;
import 'package:vibration/vibration.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final prefs = await SharedPreferences.getInstance();
  runApp(MultiTagApp(prefs: prefs));
}

class MultiTagApp extends StatelessWidget {
  final SharedPreferences prefs;
  final String serverUrl = 'ws://YOUR_SERVER_IP:8080/ws';

  MultiTagApp({required this.prefs});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'MultiTag Sync',
      theme: ThemeData(primarySwatch: Colors.blue),
      home: MainScreen(prefs: prefs, serverUrl: serverUrl),
    );
  }
}

class MainScreen extends StatefulWidget {
  final SharedPreferences prefs;
  final String serverUrl;

  MainScreen({required this.prefs, required this.serverUrl});

  @override
  _MainScreenState createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  late WebSocketChannel _channel;
  late String _clientId;
  bool _isReady = false;
  bool _isConnected = false;
  bool _isLoading = false;
  bool _showImage = false;
  int _readyCount = 0;
  int _totalCount = 0;
  String? _imageUrl;
  String? _targetTimeUTC;

  @override
  void initState() {
    super.initState();
    _initClient();
    _connectToServer();
  }

  void _initClient() {
    _clientId = widget.prefs.getString('clientId') ?? const Uuid().v4();
    widget.prefs.setString('clientId', _clientId);
  }

  void _connectToServer() {
    _channel = WebSocketChannel.connect(Uri.parse(widget.serverUrl));
    _channel.stream.listen(
      (message) => _handleServerMessage(message),
      onDone: () => _handleDisconnection(),
      onError: (error) => _handleDisconnection(),
    );

    // Send registration message
    _channel.sink.add('{"type":"register","clientId":"$_clientId"}');
    setState(() => _isConnected = true);
  }

  void _handleServerMessage(dynamic message) {
    print('Received: $message');
    try {
      final data = jsonDecode(message);
      switch (data['type']) {
        case 'full_state':
          setState(() {
            _readyCount = data['state']['readyCount'] ?? 0;
            _totalCount = data['state']['totalCount'] ?? 0;
            _imageUrl = data['state']['hasImage'] == true
              ? 'http://YOUR_SERVER_IP:8080/image'
              : null;
          });
          break;
          
        case 'partial_state':
          setState(() {
            _readyCount = data['readyCount'] ?? 0;
            _totalCount = data['totalCount'] ?? 0;
          });
          break;
          
        case 'image_updated':
          setState(() {
            _imageUrl = 'http://YOUR_SERVER_IP:8080/image?t=${DateTime.now().millisecondsSinceEpoch}';
          });
          break;
          
        case 'start':
          setState(() {
            _targetTimeUTC = data['targetTimestampUTC'];
          });
          _scheduleSyncActions();
          break;
      }
    } catch (e) {
      print('Error handling message: $e');
    }
  }

  void _scheduleSyncActions() async {
    if (_targetTimeUTC == null) return;
    
    final targetTime = DateTime.parse(_targetTimeUTC!);
    final now = DateTime.now().toUtc();
    final delay = targetTime.difference(now);
    
    if (delay.isNegative) return; // Already past the target time
    
    await Future.delayed(delay);
    
    // 1. Vibrate
    if (await Vibration.hasVibrator()) {
      Vibration.vibrate(duration: 500);
    }
    
    // 2. Show loading screen for 3 seconds
    setState(() => _isLoading = true);
    await Future.delayed(Duration(seconds: 3));
    setState(() => _isLoading = false);
    
    // 3. Show image
    setState(() => _showImage = true);
  }

  void _handleDisconnection() {
    setState(() => _isConnected = false);
    // TODO: Implement reconnection logic
  }

  void _toggleReady() {
    setState(() => _isReady = !_isReady);
    _channel.sink.add('{"type":"ready","clientId":"$_clientId","isReady":$_isReady}');
  }

  Future<void> _uploadImage() async {
    final picker = ImagePicker();
    final pickedFile = await picker.pickImage(source: ImageSource.gallery);
    if (pickedFile == null) return;

    final uri = Uri.parse('http://YOUR_SERVER_IP:8080/upload');
    final request = http.MultipartRequest('POST', uri);
    request.files.add(await http.MultipartFile.fromPath(
      'image',
      pickedFile.path,
    ));

    try {
      final response = await request.send();
      if (response.statusCode == 200) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Image uploaded successfully')),
        );
      }
    } catch (e) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Failed to upload image: $e')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('MultiTag Sync')),
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            if (_isLoading)
              CircularProgressIndicator()
            else if (_showImage && _imageUrl != null)
              Expanded(
                child: FutureBuilder(
                  future: DefaultCacheManager().getSingleFile(_imageUrl!),
                  builder: (context, snapshot) {
                    if (snapshot.hasData) {
                      return Image.file(snapshot.data!);
                    }
                    return Center(child: CircularProgressIndicator());
                  },
                )
              )
            else ...[
              Text('Status: ${_isConnected ? 'Connected' : 'Disconnected'}'),
              Text('Ready: $_readyCount/$_totalCount'),
              ElevatedButton(
                onPressed: _toggleReady,
                child: Text(_isReady ? 'Not Ready' : 'I\'m Ready'),
              ),
              // Admin controls
              if (kDebugMode) ...[
                SizedBox(height: 20),
                ElevatedButton(
                  onPressed: _uploadImage,
                  child: Text('Upload Image (Admin)'),
                ),
              ],
            ],
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _channel.sink.close();
    super.dispose();
  }
}
