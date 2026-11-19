import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'models.dart';

/// 本地持久化：登录态、服务地址、按会话的消息缓存（离线可读）。
class Store {
  static late SharedPreferences _sp;

  static Future<void> init() async {
    _sp = await SharedPreferences.getInstance();
  }

  static String getStr(String key) => _sp.getString(key) ?? '';
  static Future<void> setStr(String key, String value) => _sp.setString(key, value);
  static Future<void> remove(String key) => _sp.remove(key);

  static String get deviceId {
    var id = getStr('device_id');
    if (id.isEmpty) {
      id = 'mobile-${DateTime.now().millisecondsSinceEpoch.toRadixString(36)}';
      _sp.setString('device_id', id);
    }
    return id;
  }

  static Future<void> saveMessages(String convId, List<Msg> list) async {
    final tail = list.length > 120 ? list.sublist(list.length - 120) : list;
    await _sp.setString('msg_$convId', jsonEncode(tail.map((m) => m.toCache()).toList()));
  }

  static List<Msg> readMessages(String convId, String myUid) {
    final raw = getStr('msg_$convId');
    if (raw.isEmpty) return [];
    try {
      final list = jsonDecode(raw);
      if (list is! List) return [];
      return list
          .whereType<Map>()
          .map((j) => Msg.fromJson(Map<String, dynamic>.from(j), myUid: myUid))
          .toList();
    } catch (_) {
      return [];
    }
  }

  static Future<void> clearMessages(String convId) => _sp.remove('msg_$convId');
}
