import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../core/models.dart';
import '../widgets/avatar.dart';

/// 添加外部联系人：搜索用户、发送好友申请、处理收到的好友申请
class AddContactPage extends StatefulWidget {
  const AddContactPage({super.key});

  @override
  State<AddContactPage> createState() => _AddContactPageState();
}

class _AddContactPageState extends State<AddContactPage> {
  final _search = TextEditingController();
  List<Contact> _users = [];
  Map<String, Contact> _userMap = {};
  Set<String> _friendUids = {};
  List<Map<String, dynamic>> _applies = [];
  final Set<String> _applied = {};
  bool _loading = true;
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  Future<void> _load({String keyword = ''}) async {
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      final myUid = context.read<AppState>().myUid;
      final q = keyword.trim();
      final results = await Future.wait([
        Api.get('/users'),
        Api.get('/friends'),
        Api.get('/friends/applies'),
        if (q.isNotEmpty) Api.get('/users/search?q=${Uri.encodeComponent(q)}'),
      ]);
      List<Contact> parse(dynamic data) => (data is List)
          ? data.whereType<Map>().map((j) => Contact.fromJson(Map<String, dynamic>.from(j))).toList()
          : <Contact>[];
      final all = parse(results[0]);
      final friends = parse(results[1]);
      final applies = (results[2] is List)
          ? (results[2] as List).whereType<Map>().map((j) => Map<String, dynamic>.from(j)).toList()
          : <Map<String, dynamic>>[];
      _users = (q.isEmpty ? all : parse(results[3])).where((u) => u.uid.isNotEmpty && u.uid != myUid).toList();
      _friendUids = friends.map((f) => f.uid).toSet();
      _userMap = {for (final u in all) u.uid: u};
      _applies = applies;
    } catch (e) {
      _error = e is ApiException ? e.message : '$e';
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _apply(Contact u) async {
    if (_applied.contains(u.uid)) return;
    setState(() => _applied.add(u.uid));
    try {
      await Api.post('/friends/apply', {'toUid': u.uid, 'message': '你好，我想添加你为联系人'});
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('已向 ${u.nickname} 发送好友申请')));
      }
    } catch (e) {
      if (mounted) {
        setState(() => _applied.remove(u.uid));
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e is ApiException ? e.message : '好友申请发送失败')),
        );
      }
    }
  }

  Future<void> _handleApply(Map<String, dynamic> a, bool accept) async {
    final id = a['id'];
    try {
      await Api.put('/friends/applies/$id', {'accept': accept});
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(accept ? '已添加为联系人' : '已拒绝该申请')));
      await _load(keyword: _search.text);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e is ApiException ? e.message : '操作失败')),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      appBar: AppBar(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.white,
        title: const Text('添加外部联系人', style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700)),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(58),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(14, 0, 14, 10),
            child: TextField(
              controller: _search,
              textInputAction: TextInputAction.search,
              onSubmitted: (v) => _load(keyword: v),
              decoration: InputDecoration(
                isDense: true,
                hintText: '搜索姓名、账号或手机号',
                hintStyle: const TextStyle(fontSize: 14, color: Color(0xFFADB3C1)),
                prefixIcon: const Icon(Icons.search, size: 20),
                suffixIcon: _search.text.isEmpty
                    ? null
                    : IconButton(
                        icon: const Icon(Icons.close, size: 18),
                        onPressed: () {
                          _search.clear();
                          _load();
                        },
                      ),
                filled: true,
                fillColor: const Color(0xFFF5F6FA),
                contentPadding: const EdgeInsets.symmetric(vertical: 11),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide.none,
                ),
              ),
            ),
          ),
        ),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error.isNotEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(_error, style: const TextStyle(color: Colors.grey)),
                      const SizedBox(height: 10),
                      FilledButton(onPressed: () => _load(keyword: _search.text), child: const Text('重试')),
                    ],
                  ),
                )
              : RefreshIndicator(
                  onRefresh: () => _load(keyword: _search.text),
                  child: ListView(
                    padding: const EdgeInsets.only(bottom: 24),
                    children: [
                      if (_applies.isNotEmpty) ...[
                        _section('新的朋友'),
                        for (final a in _applies) _applyRow(a),
                      ],
                      _section(_search.text.trim().isEmpty ? '全部用户' : '搜索结果'),
                      if (_users.isEmpty)
                        const Padding(
                          padding: EdgeInsets.symmetric(vertical: 40),
                          child: Center(
                            child: Text('没有找到相关用户', style: TextStyle(color: Color(0xFF8A95AA))),
                          ),
                        ),
                      for (final u in _users) _userRow(u),
                    ],
                  ),
                ),
    );
  }

  Widget _section(String title) => Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 6),
        child: Text(title,
            style: const TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA), fontWeight: FontWeight.w600)),
      );

  Widget _applyRow(Map<String, dynamic> a) {
    final fromUid = '${a['fromUid'] ?? ''}';
    final u = _userMap[fromUid];
    final nick = '${a['nickname'] ?? ''}';
    final name = nick.isNotEmpty ? nick : (u?.nickname.isNotEmpty == true ? u!.nickname : fromUid);
    final avatar = '${a['avatar'] ?? ''}'.isNotEmpty ? '${a['avatar']}' : (u?.avatar ?? '');
    final msg = '${a['message'] ?? ''}';
    return ListTile(
      leading: Avatar(name: name, url: avatar, size: 44),
      title: Text(name, style: const TextStyle(fontSize: 15.5, fontWeight: FontWeight.w600)),
      subtitle: Text(msg.isEmpty ? '请求添加你为联系人' : msg,
          maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 12.5)),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          TextButton(
            onPressed: () => _handleApply(a, false),
            child: const Text('拒绝', style: TextStyle(color: Color(0xFF8A95AA))),
          ),
          const SizedBox(width: 2),
          FilledButton(
            style: FilledButton.styleFrom(
              minimumSize: const Size(58, 32),
              padding: const EdgeInsets.symmetric(horizontal: 12),
            ),
            onPressed: () => _handleApply(a, true),
            child: const Text('同意', style: TextStyle(fontSize: 13)),
          ),
        ],
      ),
    );
  }

  Widget _userRow(Contact u) {
    final isFriend = _friendUids.contains(u.uid);
    final isApplied = _applied.contains(u.uid);
    return ListTile(
      leading: Avatar(name: u.nickname, url: u.avatar, size: 44),
      title: Row(
        children: [
          Flexible(
            child: Text(u.nickname,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 15.5, fontWeight: FontWeight.w600)),
          ),
          if (u.external) ...[
            const SizedBox(width: 6),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
              decoration: BoxDecoration(color: const Color(0xFFFFF3EC), borderRadius: BorderRadius.circular(4)),
              child: const Text('外部',
                  style: TextStyle(fontSize: 10, color: Color(0xFFFF7A45), fontWeight: FontWeight.w600)),
            ),
          ],
        ],
      ),
      subtitle: Text(
        [u.deptName, u.title, '@${u.username}'].where((s) => s.isNotEmpty).join(' · '),
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA)),
      ),
      trailing: isFriend
          ? const Text('已是好友', style: TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA)))
          : isApplied
              ? const Text('已申请', style: TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA)))
              : OutlinedButton(
                  style: OutlinedButton.styleFrom(
                    minimumSize: const Size(58, 32),
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                  ),
                  onPressed: () => _apply(u),
                  child: const Text('添加', style: TextStyle(fontSize: 13)),
                ),
    );
  }
}
