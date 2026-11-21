import 'dart:convert';

const Map<int, String> kMsgTypes = {
  1: 'text',
  2: 'image',
  3: 'system',
  4: 'voice',
  5: 'file',
  6: 'video',
  7: 'call',
};

String normType(dynamic v) {
  if (v is int) return kMsgTypes[v] ?? 'text';
  return (v == null || '$v'.isEmpty) ? 'text' : '$v';
}

dynamic parseExtra(dynamic v) {
  if (v == null) return null;
  if (v is Map) return Map<String, dynamic>.from(v);
  final s = '$v'.trim();
  if (s.isEmpty) return null;
  try {
    final d = jsonDecode(s);
    return d is Map ? Map<String, dynamic>.from(d) : null;
  } catch (_) {
    return null;
  }
}

class Conv {
  String convId;
  int convType;
  String peerId;
  String peerName;
  String peerAvatar;
  int unread;
  String lastMsg;
  int lastMsgAt;
  bool pinned;
  bool muted;

  Conv({
    required this.convId,
    this.convType = 1,
    this.peerId = '',
    this.peerName = '',
    this.peerAvatar = '',
    this.unread = 0,
    this.lastMsg = '',
    this.lastMsgAt = 0,
    this.pinned = false,
    this.muted = false,
  });

  factory Conv.fromJson(Map<String, dynamic> j) => Conv(
        convId: '${j['convId'] ?? ''}',
        convType: (j['convType'] as num?)?.toInt() ?? 1,
        peerId: '${j['peerId'] ?? ''}',
        peerName: '${j['peerName'] ?? ''}',
        peerAvatar: '${j['peerAvatar'] ?? ''}',
        unread: (j['unread'] as num?)?.toInt() ?? 0,
        lastMsg: '${j['lastMsg'] ?? ''}',
        lastMsgAt: (j['lastMsgAt'] as num?)?.toInt() ?? 0,
        pinned: ((j['pinned'] as num?)?.toInt() ?? 0) == 1,
        muted: ((j['muted'] as num?)?.toInt() ?? 0) == 1,
      );

  bool get isGroup => convType == 2;
  String get title => peerName.isNotEmpty ? peerName : peerId;
}

class Msg {
  String msgId;
  String clientMsgId;
  String convId;
  int seq;
  String fromUid;
  String fromName;
  String type;
  String content;
  dynamic extra;
  int createdAt;
  int status;
  bool mine;
  bool pending;

  Msg({
    required this.msgId,
    this.clientMsgId = '',
    this.convId = '',
    this.seq = 0,
    this.fromUid = '',
    this.fromName = '',
    this.type = 'text',
    this.content = '',
    this.extra,
    this.createdAt = 0,
    this.status = 1,
    this.mine = false,
    this.pending = false,
  });

  factory Msg.fromJson(Map<String, dynamic> j, {String myUid = '', bool markMine = true}) {
    final fromUid = '${j['fromUid'] ?? ''}';
    return Msg(
      msgId: '${j['msgId'] ?? ''}',
      clientMsgId: '${j['clientMsgId'] ?? ''}',
      convId: '${j['convId'] ?? ''}',
      seq: (j['seq'] as num?)?.toInt() ?? 0,
      fromUid: fromUid,
      fromName: '${j['fromName'] ?? ''}',
      type: normType(j['type']),
      content: '${j['content'] ?? ''}',
      extra: parseExtra(j['extra']),
      createdAt: (j['createdAt'] as num?)?.toInt() ?? 0,
      status: (j['status'] as num?)?.toInt() ?? 1,
      mine: markMine && fromUid.isNotEmpty && fromUid == myUid,
      pending: false,
    );
  }

  Map<String, dynamic> toCache() => {
        'msgId': msgId,
        'clientMsgId': clientMsgId,
        'convId': convId,
        'seq': seq,
        'fromUid': fromUid,
        'fromName': fromName,
        'type': type,
        'content': content,
        'extra': extra == null ? null : jsonEncode(extra),
        'createdAt': createdAt,
        'status': status,
      };
}

class Contact {
  final String uid;
  final String username;
  final String nickname;
  final String avatar;
  final String phone;
  final String email;
  final String deptName;
  final String orgName;
  final String title;
  final bool external;

  Contact.fromJson(Map<String, dynamic> j)
      : uid = '${j['uid'] ?? j['friendUid'] ?? ''}',
        username = '${j['username'] ?? ''}',
        nickname = '${j['nickname'] ?? j['remark'] ?? j['uid'] ?? ''}',
        avatar = '${j['avatar'] ?? ''}',
        phone = '${j['phone'] ?? ''}',
        email = '${j['email'] ?? ''}',
        deptName = '${j['deptName'] ?? ''}',
        orgName = '${j['orgName'] ?? ''}',
        title = '${j['title'] ?? ''}',
        external = j['external'] == true || (j['deptId'] as num?)?.toInt() == 0;
}

class GroupInfo {
  final String groupId;
  final String name;
  final int memberCount;
  final String notice;

  GroupInfo.fromJson(Map<String, dynamic> j)
      : groupId = '${j['groupId'] ?? ''}',
        name = '${j['name'] ?? ''}',
        memberCount = (j['memberCount'] as num?)?.toInt() ?? 0,
        notice = '${j['notice'] ?? ''}';
}
