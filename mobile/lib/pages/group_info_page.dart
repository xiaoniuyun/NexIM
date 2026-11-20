import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../widgets/avatar.dart';
import 'friend_picker_page.dart';

/// 群聊信息：成员列表、邀请、移出、退出、解散
class GroupInfoPage extends StatefulWidget {
  final String groupId;
  final String groupName;
  const GroupInfoPage({super.key, required this.groupId, required this.groupName});

  @override
  State<GroupInfoPage> createState() => _GroupInfoPageState();
}

class _GroupInfoPageState extends State<GroupInfoPage> {
  Map<String, dynamic> _group = {};
  List<Map<String, dynamic>> _members = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final results = await Future.wait([
        Api.get('/groups/${widget.groupId}'),
        Api.get('/groups/${widget.groupId}/members'),
      ]);
      if (!mounted) return;
      setState(() {
        _group = results[0] is Map ? Map<String, dynamic>.from(results[0] as Map) : {};
        _members = (results[1] is List)
            ? (results[1] as List).whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList()
            : [];
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _loading = false);
      _toast('加载失败：$e');
    }
  }

  String get _name => '${_group['name'] ?? widget.groupName}';
  String get _ownerUid => '${_group['ownerUid'] ?? ''}';
  bool get _isOwner => _ownerUid.isNotEmpty && _ownerUid == context.read<AppState>().myUid;

  void _toast(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _invite() async {
    final exist = _members.map((m) => '${m['uid'] ?? ''}').toSet();
    final picked = await Navigator.of(context).push<List<String>>(
      MaterialPageRoute(builder: (_) => FriendPickerPage(title: '邀请成员', disabled: exist)),
    );
    if (picked == null || picked.isEmpty || !mounted) return;
    try {
      await Api.post('/groups/${widget.groupId}/invite', {'members': picked});
      if (!mounted) return;
      _toast('已邀请 ${picked.length} 位成员');
      _load();
    } catch (e) {
      if (mounted) _toast('邀请失败：$e');
    }
  }

  Future<void> _removeMember(Map<String, dynamic> m) async {
    final uid = '${m['uid'] ?? ''}';
    final name = '${m['nickname'] ?? uid}';
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('移出成员'),
        content: Text('将「$name」移出群聊？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('移出', style: TextStyle(color: Color(0xFFE5484D))),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    try {
      await Api.delete('/groups/${widget.groupId}/members/${Uri.encodeComponent(uid)}');
      if (!mounted) return;
      _toast('已移出成员');
      _load();
    } catch (e) {
      if (mounted) _toast('操作失败：$e');
    }
  }

  Future<void> _leave() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('退出群聊'),
        content: const Text('退出后将不再接收该群消息，确认退出？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('退出', style: TextStyle(color: Color(0xFFE5484D))),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    try {
      await Api.delete('/groups/${widget.groupId}/members/${Uri.encodeComponent(context.read<AppState>().myUid)}');
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (mounted) _toast('退出失败：$e');
    }
  }

  Future<void> _dissolve() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('解散群聊'),
        content: const Text('解散后所有成员将无法继续在群内发言，确认解散？'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('解散', style: TextStyle(color: Color(0xFFE5484D))),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    try {
      await Api.delete('/groups/${widget.groupId}');
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      if (mounted) _toast('解散失败：$e');
    }
  }

  @override
  Widget build(BuildContext context) {
    final me = context.read<AppState>().myUid;
    return Scaffold(
      backgroundColor: const Color(0xFFF5F6FA),
      appBar: AppBar(
        backgroundColor: Colors.white,
        elevation: 0,
        centerTitle: true,
        title: const Text('群聊信息',
            style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700, color: Color(0xFF1B2333))),
        iconTheme: const IconThemeData(color: Color(0xFF1B2333)),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              children: [
                Container(
                  color: Colors.white,
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 18),
                  child: Row(
                    children: [
                      Avatar(name: _name, url: '${_group['avatar'] ?? ''}', size: 54),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(_name,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(
                                    fontSize: 17, fontWeight: FontWeight.w700, color: Color(0xFF1B2333))),
                            const SizedBox(height: 4),
                            Text('群成员 ${_members.length} 人',
                                style: const TextStyle(fontSize: 12.5, color: Color(0xFF9AA5B8))),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 10),
                Container(
                  color: Colors.white,
                  child: Column(
                    children: [
                      Padding(
                        padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
                        child: Row(
                          children: [
                            const Text('群成员',
                                style: TextStyle(fontWeight: FontWeight.w700, color: Color(0xFF1B2333))),
                            const Spacer(),
                            TextButton.icon(
                              onPressed: _invite,
                              icon: const Icon(Icons.person_add_alt_1, size: 18),
                              label: const Text('邀请成员'),
                            ),
                          ],
                        ),
                      ),
                      for (final m in _members)
                        ListTile(
                          leading: Avatar(
                              name: '${m['nickname'] ?? m['uid']}',
                              url: '${m['avatar'] ?? ''}',
                              size: 40),
                          title: Row(
                            children: [
                              Flexible(
                                child: Text('${m['nickname'] ?? m['uid']}',
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: const TextStyle(fontSize: 15)),
                              ),
                              if ('${m['role'] ?? ''}' == 'owner')
                                Container(
                                  margin: const EdgeInsets.only(left: 6),
                                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                                  decoration: BoxDecoration(
                                      color: const Color(0xFFFFF3E0), borderRadius: BorderRadius.circular(4)),
                                  child: const Text('群主',
                                      style: TextStyle(fontSize: 11, color: Color(0xFFE08A00))),
                                ),
                              if ('${m['uid'] ?? ''}' == me)
                                const Text(' 我', style: TextStyle(fontSize: 12, color: Color(0xFF9AA5B8))),
                            ],
                          ),
                          trailing: (_isOwner && '${m['uid'] ?? ''}' != me && '${m['role'] ?? ''}' != 'owner')
                              ? TextButton(
                                  onPressed: () => _removeMember(m),
                                  child: const Text('移出',
                                      style: TextStyle(fontSize: 13, color: Color(0xFFE5484D))),
                                )
                              : null,
                        ),
                      const SizedBox(height: 6),
                    ],
                  ),
                ),
                const SizedBox(height: 10),
                Container(
                  color: Colors.white,
                  child: Column(
                    children: [
                      ListTile(
                        title: const Center(
                            child: Text('退出群聊',
                                style: TextStyle(color: Color(0xFFE5484D), fontWeight: FontWeight.w600))),
                        onTap: _leave,
                      ),
                      if (_isOwner)
                        ListTile(
                          title: const Center(
                              child: Text('解散群聊',
                                  style: TextStyle(color: Color(0xFFE5484D), fontWeight: FontWeight.w600))),
                          onTap: _dissolve,
                        ),
                    ],
                  ),
                ),
              ],
            ),
    );
  }
}
