import 'package:flutter_cache_manager/flutter_cache_manager.dart';

/// 图片磁盘缓存：保存在应用缓存目录（/data/user/0/<包名>/cache/nexim_images），
/// 7 天过期、最多 200 张，重启后仍可直接展示已看过的图片。
class AppImages {
  static final CacheManager cache = CacheManager(
    Config(
      'nexim_images',
      stalePeriod: const Duration(days: 7),
      maxNrOfCacheObjects: 200,
    ),
  );
}
