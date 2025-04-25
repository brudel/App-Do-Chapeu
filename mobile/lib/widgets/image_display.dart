import 'package:flutter/material.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

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
        future: DefaultCacheManager().getSingleFile(imageUrl),
        builder: (context, snapshot) {
          if (snapshot.hasData) {
            return Image.file(snapshot.data!);
          }
          return const Center(child: CircularProgressIndicator());
        },
      ),
    );
  }
}