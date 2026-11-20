import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import '../core/api.dart';
import '../core/image_cache.dart';
import '../core/util.dart';

class Avatar extends StatelessWidget {
  final String name;
  final String url;
  final double size;

  const Avatar({super.key, required this.name, this.url = '', this.size = 44});

  @override
  Widget build(BuildContext context) {
    final label = name.isEmpty ? 'U' : name.substring(0, 1);
    // 后端头像字段 file:<fileId> 表示上传的图片，其余（颜色等历史值）按首字母渲染
    final src = url.startsWith('file:') ? Api.fileUrl(url.substring(5)) : url;
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
    if (src.isEmpty) return fallback;
    return ClipRRect(
      borderRadius: BorderRadius.circular(size * 0.28),
      child: CachedNetworkImage(
        imageUrl: src,
        // token 会刷新，缓存键固定为不带 query 的地址，避免重复下载
        cacheKey: src.split('?').first,
        cacheManager: AppImages.cache,
        width: size,
        height: size,
        fit: BoxFit.cover,
        placeholder: (_, __) => fallback,
        errorWidget: (_, __, ___) => fallback,
      ),
    );
  }
}
