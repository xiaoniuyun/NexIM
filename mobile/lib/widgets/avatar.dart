import 'package:flutter/material.dart';

import '../core/util.dart';

class Avatar extends StatelessWidget {
  final String name;
  final String url;
  final double size;

  const Avatar({super.key, required this.name, this.url = '', this.size = 44});

  @override
  Widget build(BuildContext context) {
    final label = name.isEmpty ? 'U' : name.substring(0, 1);
    final fallback = Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: Color(avatarColor(name)),
        borderRadius: BorderRadius.circular(size * 0.28),
      ),
      child: Text(
        label,
        style: TextStyle(color: Colors.white, fontSize: size * 0.42, fontWeight: FontWeight.w600),
      ),
    );
    if (url.isEmpty) return fallback;
    return ClipRRect(
      borderRadius: BorderRadius.circular(size * 0.28),
      child: Image.network(
        url,
        width: size,
        height: size,
        fit: BoxFit.cover,
        errorBuilder: (_, __, ___) => fallback,
        loadingBuilder: (c, w, p) => p == null ? w : fallback,
      ),
    );
  }
}
