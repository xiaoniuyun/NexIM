import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

class ApiException implements Exception {
  final String message;
  ApiException(this.message);
  @override
  String toString() => message;
}

/// 极简 REST 客户端：Bearer 鉴权、2001 自动刷新、multipart 上传。
class Api {
  static String base = 'http://127.0.0.1:8080/api/v1';
  static String token = '';
  static String refreshToken = '';
  static void Function()? onKicked;

  static Map<String, String> _headers({bool json = true}) => {
        if (json) 'Content-Type': 'application/json',
        if (token.isNotEmpty) 'Authorization': 'Bearer $token',
      };

  static Uri _uri(String path) => Uri.parse(base + path);

  static Future<dynamic> get(String path) => _send('GET', path);
  static Future<dynamic> post(String path, [Map<String, dynamic>? body]) =>
      _send('POST', path, body: body);
  static Future<dynamic> put(String path, [Map<String, dynamic>? body]) =>
      _send('PUT', path, body: body);
  static Future<dynamic> patch(String path, [Map<String, dynamic>? body]) =>
      _send('PATCH', path, body: body);
  static Future<dynamic> delete(String path) => _send('DELETE', path);

  static Future<dynamic> _send(String method, String path,
      {Map<String, dynamic>? body, bool retry = true}) async {
    final req = http.Request(method, _uri(path));
    req.headers.addAll(_headers());
    if (body != null) req.body = jsonEncode(body);
    final res = await http.Response.fromStream(await req.send());
    dynamic parsed;
    try {
      parsed = jsonDecode(utf8.decode(res.bodyBytes));
    } catch (_) {
      parsed = <String, dynamic>{};
    }
    if (parsed is Map && parsed['code'] == 2001 && refreshToken.isNotEmpty && retry) {
      if (await _refresh()) return _send(method, path, body: body, retry: false);
    }
    if (parsed is Map && parsed['code'] == 2001) {
      onKicked?.call();
      throw ApiException('登录已过期，请重新登录');
    }
    if (res.statusCode >= 400) {
      throw ApiException(
          (parsed is Map && parsed['message'] != null) ? '${parsed['message']}' : '请求失败 ${res.statusCode}');
    }
    if (parsed is Map) {
      final code = parsed['code'];
      if (code is int && code != 0) throw ApiException('${parsed['message'] ?? '请求失败'}');
      return parsed['data'];
    }
    return parsed;
  }

  static Future<bool> _refresh() async {
    try {
      final res = await http.post(_uri('/auth/refresh'),
          headers: {'Content-Type': 'application/json'},
          body: jsonEncode({'refreshToken': refreshToken}));
      final d = jsonDecode(utf8.decode(res.bodyBytes));
      if (d is Map && d['code'] == 0) {
        token = d['data']['accessToken'] ?? '';
        refreshToken = d['data']['refreshToken'] ?? '';
        return token.isNotEmpty;
      }
    } catch (_) {}
    return false;
  }

  static Future<dynamic> upload(String path, String filePath,
      {String field = 'file', Map<String, String>? fields}) async {
    final req = http.MultipartRequest('POST', _uri(path));
    req.headers.addAll(_headers(json: false));
    if (fields != null) req.fields.addAll(fields);
    req.files.add(await http.MultipartFile.fromPath(field, filePath));
    final res = await http.Response.fromStream(await req.send());
    dynamic parsed;
    try {
      parsed = jsonDecode(utf8.decode(res.bodyBytes));
    } catch (_) {
      parsed = <String, dynamic>{};
    }
    if (parsed is Map && parsed['code'] == 0) return parsed['data'];
    throw ApiException((parsed is Map && parsed['message'] != null) ? '${parsed['message']}' : '上传失败');
  }

  static String fileUrl(String fileId) =>
      '$base/files/${Uri.encodeComponent(fileId)}/download?token=${Uri.encodeComponent(token)}';

  /// 由 API 地址推导 WebSocket 地址（默认 HTTP 端口 +1，路径 /ws）
  static String wsUrl({String device = '', String platform = 'Android'}) {
    final u = Uri.parse(base.replaceAll(RegExp(r'/api/v1/?$'), ''));
    final wsScheme = u.scheme == 'https' ? 'wss' : 'ws';
    final port = u.hasPort && u.port > 0 ? u.port + 1 : 8081;
    final uri = Uri(
      scheme: wsScheme,
      host: u.host,
      port: port,
      path: '/ws',
      queryParameters: {'token': token, 'device': device, 'platform': platform},
    );
    return uri.toString();
  }

  static String platformName() => Platform.isIOS ? 'iOS' : 'Android';
}
