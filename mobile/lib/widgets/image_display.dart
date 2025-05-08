import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

class ImageDisplay extends StatelessWidget {
  final String imageUrl;

  const ImageDisplay({
    required this.imageUrl,
    Key? key,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child: FutureBuilder(
        future: http.get(Uri.parse('http://$imageUrl/image')),
        builder: (context, snapshot) {
          if (snapshot.hasData) {
            return Image.memory(snapshot.data!.bodyBytes);
          }
          return const Center(child: CircularProgressIndicator());
        },
      ),
    );
  }
}