import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:flutter/foundation.dart';

import '../core/api.dart';
import '../core/models.dart';
import '../core/store.dart';
import '../core/ws_client.dart';

/// 全局业务状态：登录、会话、消息、长连接（对齐 H5 stores/chat.js 的协议）。
class AppState extends ChangeNotifier {
  final WsClient ws = WsClient();
  final _callSignals = StreamController<Map<String, dynamic>>.broadcast();
  Stream<Map<String, dynamic>> get callSignals => _callSignals.stream;

  Map<String, dynamic>? user;
  String deviceId = '';
  String apiBase = '';
  List<Conv> convs = [];
  List<Msg> messages = [];
  String currentConvId = '';
  // 是否停留在聊天页：只有正在查看时才自动回执已读
  bool chatVisible = false;
  bool connected = false;
  String lastError = '';

  String get myUid => '${user?['uid'] ?? ''}';
  String get myName => '${user?['nickname'] ?? user?['username'] ?? ''}';
  bool get isLogin => user != null && Api.token.isNotEmpty;
  int get unreadTotal => convs.fold(0, (s, c) => s + c.unread);

  Future<void> init() async {
    await Store.init();
    apiBase = Store.getStr('api_base');
    if (apiBase.isEmpty) {
      apiBase = Platform.isAndroid
          ? 'http://10.0.2.2:8080/api/v1'
          : 'http://127.0.0.1:8080/api/v1';
      await Store.setStr('api_base', apiBase);
    }
    Api.base = apiBase;
    Api.token = Store.getStr('token');
    Api.refreshToken = Store.getStr('refresh');
    deviceId = Store.deviceId;
    final rawUser = Store.getStr('user');
    if (rawUser.isNotEmpty) {
      try {
        user = Map<String, dynamic>.from(jsonDecode(rawUser));
      } catch (_) {}
    }
    Api.onKicked = logoutLocal;
    ws.onState = (v) {
      connected = v;
      notifyListeners();
      if (v) {
        ws.send('AUTH', {});
        loadConvs();
        if (currentConvId.isNotEmpty) fetchMessages(currentConvId);
      }
    };
    ws.onFrame = _handleFrame;
    if (isLogin) {
      loadConvs();
      connect();
    }
  }

  Future<void> setServer(String url) async {
    var v = url.trim();
    if (v.isEmpty) return;
    v = v.replaceAll(RegExp(r'/+$'), '');
    if (!v.endsWith('/api/v1')) v = '$v/api/v1';
    apiBase = v;
    Api.base = v;
    await Store.setStr('api_base', v);
    notifyListeners();
  }

  Future<void> login(String account, String password) async {
    final data = await Api.post('/auth/login', {
      'account': account,
      'password': password,
      'deviceId': deviceId,
      'platform': Platform.isIOS ? 'iOS' : 'Android',
    });
    Api.token = '${data['accessToken'] ?? ''}';
    Api.refreshToken = '${data['refreshToken'] ?? ''}';
    user = {
      'uid': data['uid'],
      'username': data['username'],
      'nickname': data['nickname'],
      'avatar': data['avatar'],
      'signature': data['signature'] ?? '',
      'gender': data['gender'] ?? 0,
      'phone': data['phone'] ?? '',
      'title': data['title'] ?? '',
      'roles': data['roles'] ?? [],
    };
    await Store.setStr('token', Api.token);
    await Store.setStr('refresh', Api.refreshToken);
    await Store.setStr('user', jsonEncode(user));
    notifyListeners();
    connect();
    await loadConvs();
  }

  Future<void> logout() async {
    try {
      await Api.post('/auth/logout');
    } catch (_) {}
    logoutLocal();
  }

  // 合并个人资料变更（昵称/头像/性别/签名）并持久化
  Future<void> applyProfile(Map<String, dynamic> data) async {
    if (data.isEmpty) return;
    user = {...?user, ...data};
    await Store.setStr('user', jsonEncode(user));
    notifyListeners();
  }

  Future<void> updateProfile(Map<String, dynamic> patch) async {
    final data = await Api.put('/users/me', patch);
    await applyProfile(Map<String, dynamic>.from(data));
  }

  Future<void> refreshProfile() async {
    try {
      final data = await Api.get('/users/me');
      await applyProfile(Map<String, dynamic>.from(data));
    } catch (_) {}
  }

  void logoutLocal() {
    ws.close();
    Api.token = '';
    Api.refreshToken = '';
    Store.remove('token');
    Store.remove('refresh');
    Store.remove('user');
    user = null;
    convs = [];
    messages = [];
    currentConvId = '';
    connected = false;
    notifyListeners();
  }

  void connect() {
    if (!isLogin) return;
    ws.connect(Api.wsUrl(device: deviceId, platform: Platform.isIOS ? 'iOS' : 'Android'));
  }

  void _handleFrame(Map<String, dynamic> f) {
    final cmd = '${f['cmd'] ?? ''}';
    final p = f['payload'] is Map ? Map<String, dynamic>.from(f['payload']) : <String, dynamic>{};
    switch (cmd) {
      case 'PUSH':
        final convId = '${p['convId'] ?? ''}';
        final fromUid = '${p['fromUid'] ?? ''}';
        final inCurrent = convId == currentConvId;
        if (inCurrent) {
          upsert(Msg(
            msgId: '${p['msgId'] ?? ''}',
            clientMsgId: '${p['clientMsgId'] ?? ''}',
            convId: convId,
            seq: (p['seq'] as num?)?.toInt() ?? 0,
            fromUid: fromUid,
            fromName: '${p['fromName'] ?? ''}',
            type: normType(p['type']),
            content: '${p['content'] ?? ''}',
            extra: parseExtra(p['extra']),
            createdAt: (p['sendTime'] as num?)?.toInt() ?? 0,
            mine: fromUid == myUid,
          ));
          Store.saveMessages(currentConvId, messages);
        }
        if (inCurrent && chatVisible && fromUid != myUid) {
          // 用户正停留在该会话，收到新消息立即回执已读，避免角标持续累加
          final c = convs.where((x) => x.convId == convId).firstOrNull;
          if (c != null) c.unread = 0;
          notifyListeners();
          _markConvRead(convId);
        } else {
          loadConvs();
        }
        break;
      case 'SEND_ACK':
        if (f['code'] == 0 && p['clientMsgId'] != null) {
          applyAck('${p['clientMsgId']}', '${p['msgId'] ?? ''}',
              (p['seq'] as num?)?.toInt() ?? 0, (p['sendTime'] as num?)?.toInt() ?? 0);
        } else if (f['code'] != 0) {
          final m = messages.reversed.where((x) => x.mine && x.pending).firstOrNull;
          if (m != null) {
            m.status = 3;
            m.pending = false;
          }
          lastError = '${f['message'] ?? '消息发送失败'}';
          notifyListeners();
        }
        break;
      case 'REVOKE_PUSH':
        final m = messages.where((x) => x.msgId == '${p['msgId'] ?? ''}').firstOrNull;
        if (m != null) m.status = 2;
        notifyListeners();
        loadConvs();
        break;
      case 'NOTICE_PUSH':
      case 'PRESENCE_PUSH':
        loadConvs();
        break;
      case 'KICK':
        logoutLocal();
        break;
      default:
        if (cmd.startsWith('CALL_')) _callSignals.add(f);
    }
  }

  Future<void> loadConvs() async {
    if (!isLogin) return;
    try {
      final data = await Api.get('/conversations');
      final list = (data is List)
          ? data.whereType<Map>().map((j) => Conv.fromJson(Map<String, dynamic>.from(j))).toList()
          : <Conv>[];
      list.sort((a, b) {
        if (a.pinned != b.pinned) return a.pinned ? -1 : 1;
        return b.lastMsgAt.compareTo(a.lastMsgAt);
      });
      convs = list;
      notifyListeners();
    } catch (_) {}
  }

  Future<void> openConv(String convId) async {
    currentConvId = convId;
    messages = Store.readMessages(convId, myUid);
    notifyListeners();
    await fetchMessages(convId);
    _markConvRead(convId);
  }

  void setChatVisible(bool visible) {
    chatVisible = visible;
  }

  void _markConvRead(String convId) {
    ws.send('READ', {'convId': convId});
    unawaited(Api.post('/messages/read', {'convId': convId}).then((_) => loadConvs()).catchError((_) => null));
  }

  Future<void> fetchMessages(String convId) async {
    try {
      final data = await Api.get('/conversations/${Uri.encodeComponent(convId)}/messages');
      if (currentConvId != convId) return;
      final server = (data is List)
          ? data.whereType<Map>().map((j) => Msg.fromJson(Map<String, dynamic>.from(j), myUid: myUid)).toList()
          : <Msg>[];
      final pending = messages.where((m) {
        final keep = (m.pending || m.status == 3) && m.convId == convId;
        return keep &&
            !server.any((s) => s.msgId == m.msgId || (s.clientMsgId.isNotEmpty && s.clientMsgId == m.clientMsgId));
      });
      messages = [...server, ...pending];
      messages.sort((a, b) {
        final sa = a.seq == 0 ? 1 << 62 : a.seq;
        final sb = b.seq == 0 ? 1 << 62 : b.seq;
        return sa.compareTo(sb);
      });
      notifyListeners();
      await Store.saveMessages(convId, messages);
    } catch (_) {}
  }

  void upsert(Msg m) {
    final idx = messages.indexWhere((x) =>
        (m.msgId.isNotEmpty && x.msgId == m.msgId) ||
        (m.clientMsgId.isNotEmpty && x.msgId == m.clientMsgId));
    if (idx >= 0) {
      final old = messages[idx];
      old
        ..msgId = m.msgId.isNotEmpty ? m.msgId : old.msgId
        ..clientMsgId = m.clientMsgId
        ..seq = m.seq
        ..fromUid = m.fromUid.isNotEmpty ? m.fromUid : old.fromUid
        ..fromName = m.fromName.isNotEmpty ? m.fromName : old.fromName
        ..type = m.type
        ..content = m.content
        ..extra = m.extra
        ..createdAt = m.createdAt > 0 ? m.createdAt : old.createdAt
        ..status = m.status
        ..mine = m.mine
        ..pending = false;
    } else {
      messages.add(m);
    }
    notifyListeners();
  }

  void applyAck(String clientMsgId, String msgId, int seq, int sendTime) {
    final idx = messages.indexWhere((x) => x.msgId == clientMsgId);
    if (idx < 0) return;
    final m = messages[idx];
    if (msgId.isNotEmpty) m.msgId = msgId;
    if (seq > 0) m.seq = seq;
    if (sendTime > 0) m.createdAt = sendTime;
    m.status = 1;
    m.pending = false;
    notifyListeners();
  }

  Future<void> httpSend(Map<String, dynamic> payload) async {
    try {
      final d = await Api.post('/messages/send', payload);
      applyAck('${payload['clientMsgId']}', '${d['msgId'] ?? ''}',
          (d['seq'] as num?)?.toInt() ?? 0, (d['sendTime'] as num?)?.toInt() ?? 0);
      loadConvs();
    } catch (e) {
      final idx = messages.indexWhere((x) => x.msgId == payload['clientMsgId']);
      if (idx >= 0) {
        messages[idx].status = 3;
        messages[idx].pending = false;
      }
      lastError = '$e';
      notifyListeners();
    }
  }

  String sendPayload(String type, String content, dynamic extra) {
    final conv = convs.where((c) => c.convId == currentConvId).firstOrNull;
    final clientMsgId =
        'c_${DateTime.now().millisecondsSinceEpoch}_${Random().nextInt(99999).toRadixString(36)}';
    final payload = <String, dynamic>{
      'clientMsgId': clientMsgId,
      'convId': currentConvId,
      'convType': (conv?.isGroup ?? false) ? 'group' : 'single',
      'toUid': conv?.peerId ?? '',
      'type': type,
      'content': content,
      'extra': extra == null ? '' : jsonEncode(extra),
    };
    messages.add(Msg(
      msgId: clientMsgId,
      clientMsgId: clientMsgId,
      convId: currentConvId,
      fromUid: myUid,
      fromName: myName,
      type: type,
      content: content,
      extra: extra,
      createdAt: DateTime.now().millisecondsSinceEpoch,
      mine: true,
      pending: true,
    ));
    notifyListeners();
    Store.saveMessages(currentConvId, messages);
    if (ws.connected) {
      ws.send('SEND', payload);
      // WS 未确认时自动落 HTTP 通道补发（clientMsgId 幂等）
      Timer(const Duration(milliseconds: 1800), () {
        final idx = messages.indexWhere((x) => x.msgId == clientMsgId);
        if (idx >= 0 && messages[idx].pending) httpSend(payload);
      });
    } else {
      httpSend(payload);
    }
    return clientMsgId;
  }

  String sendText(String text) => sendPayload('text', text, null);

  String sendMedia(String type, String content, Map<String, dynamic> extra) =>
      sendPayload(type, content, extra);

  void markRevoked(String msgId) {
    final idx = messages.indexWhere((m) => m.msgId == msgId);
    if (idx >= 0) {
      messages[idx].status = 2;
      notifyListeners();
    }
  }

  void removeLocal(String msgId) {
    messages.removeWhere((m) => m.msgId == msgId);
    notifyListeners();
    Store.saveMessages(currentConvId, messages);
  }

  String ensureConv(String peerId, String peerName) {
    final convId = 'conv_${([myUid, peerId]..sort()).join('_')}';
    if (!convs.any((c) => c.convId == convId)) {
      convs.insert(
          0,
          Conv(
            convId: convId,
            convType: 1,
            peerId: peerId,
            peerName: peerName,
            lastMsgAt: DateTime.now().millisecondsSinceEpoch,
          ));
      notifyListeners();
    }
    return convId;
  }

  String ensureGroupConv(String groupId, String groupName) {
    final convId = 'conv_$groupId';
    if (!convs.any((c) => c.convId == convId)) {
      convs.insert(
          0,
          Conv(
            convId: convId,
            convType: 2,
            peerId: groupId,
            peerName: groupName,
            lastMsgAt: DateTime.now().millisecondsSinceEpoch,
          ));
      notifyListeners();
    }
    return convId;
  }

  Future<void> togglePinned(Conv c) async {
    final v = !c.pinned;
    c.pinned = v;
    notifyListeners();
    try {
      await Api.patch('/conversations/${Uri.encodeComponent(c.convId)}/settings', {'pinned': v ? 1 : 0});
    } catch (_) {
      c.pinned = !v;
      notifyListeners();
    }
  }

  Future<void> toggleMuted(Conv c) async {
    final v = !c.muted;
    c.muted = v;
    notifyListeners();
    try {
      await Api.patch('/conversations/${Uri.encodeComponent(c.convId)}/settings', {'muted': v ? 1 : 0});
    } catch (_) {
      c.muted = !v;
      notifyListeners();
    }
  }

  Future<void> deleteConv(Conv c) async {
    try {
      await Api.delete('/conversations/${Uri.encodeComponent(c.convId)}');
    } catch (_) {}
    convs.removeWhere((x) => x.convId == c.convId);
    if (currentConvId == c.convId) {
      currentConvId = '';
      messages = [];
    }
    Store.clearMessages(c.convId);
    notifyListeners();
  }

  Future<void> markAllRead() async {
    try {
      await Api.post('/conversations/read-all');
    } catch (_) {}
    for (final c in convs) {
      c.unread = 0;
    }
    notifyListeners();
  }

  @override
  void dispose() {
    ws.close();
    _callSignals.close();
    super.dispose();
  }
}
