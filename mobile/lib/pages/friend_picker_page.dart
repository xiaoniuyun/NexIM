import 'package:flutter/material.dart';

import '../core/api.dart';
import '../core/models.dart';
import '../widgets/avatar.dart';

/// 好友多选页：用于建群选人、邀请成员，返回选中的 uid 列表
class FriendPickerPage extends StatefulWidget {
  final String title;
  final Set<String> disabled;
  const FriendPickerPage({super.key, this.title = '选择成员', this.disabled = const {}});

  @override
  State<FriendPickerPage> createState() => _FriendPickerPageState();
}

class _FriendPickerPageState extends State<FriendPickerPage> {
  List<Contact> friends = [];
  final Set<String> _picked = {};
  String _keyword = '';
  bool _loading = true;
  String _error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await Api.get('/friends');
      final list = (data is List)
          ? data.whereType<Map>().map((j) => Contact.fromJson(Map<String, dynamic>.from(j))).toList()
          : <Contact>[];
      if (!mounted) return;
      setState(() {
        friends = list.where((f) => f.uid.isNotEmpty && !widget.disabled.contains(f.uid)).toList();
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = '$e';
        _loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final list = _keyword.isEmpty
        ? friends
        : friends.where((f) => f.nickname.toLowerCase().contains(_keyword)).toList();
    return Scaffold(
      backgroundColor: Colors.white,
      appBar: AppBar(
        backgroundColor: Colors.white,
        elevation: 0,
        centerTitle: true,
        title: Text(widget.title,
            style: const TextStyle(fontSize: 17, fontWeight: FontWeight.w700, color: Color(0xFF1B2333))),
        iconTheme: const IconThemeData(color: Color(0xFF1B2333)),
        actions: [
          TextButton(
            onPressed: _picked.isEmpty ? null : () => Navigator.of(context).pop(_picked.toList()),
            child: Text(_picked.isEmpty ? '确定' : '确定(${_picked.length})',
                style: const TextStyle(fontWeight: FontWeight.w600)),
          ),
        ],
      ),
      body: Column(
        children: [
          Container(
            height: 42,
            margin: const EdgeInsets.fromLTRB(14, 4, 14, 8),
            padding: const EdgeInsets.symmetric(horizontal: 14),
            decoration: BoxDecoration(color: const Color(0xFFF5F6FA), borderRadius: BorderRadius.circular(21)),
            child: Row(
              children: [
                const Icon(Icons.search, size: 18, color: Color(0xFFADB3C1)),
                const SizedBox(width: 8),
                Expanded(
                  child: TextField(
                    onChanged: (v) => setState(() => _keyword = v.trim().toLowerCase()),
                    style: const TextStyle(fontSize: 14),
                    decoration: const InputDecoration(
                      isDense: true,
                      border: InputBorder.none,
                      hintText: '搜索好友',
                      hintStyle: TextStyle(fontSize: 14, color: Color(0xFFADB3C1)),
                    ),
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : _error.isNotEmpty
                    ? Center(child: Text('加载失败：$_error', style: const TextStyle(color: Color(0xFF9AA5B8))))
                    : list.isEmpty
                        ? const Center(child: Text('暂无可选好友', style: TextStyle(color: Color(0xFF9AA5B8))))
                        : ListView.builder(
                            itemCount: list.length,
                            itemBuilder: (_, i) {
                              final f = list[i];
                              final on = _picked.contains(f.uid);
                              return ListTile(
                                leading: Avatar(name: f.nickname, url: f.avatar, size: 40),
                                title: Text(f.nickname, style: const TextStyle(fontSize: 15)),
                                trailing: Checkbox(
                                  value: on,
                                  shape: const CircleBorder(),
                                  onChanged: (_) => setState(() {
                                    if (on) {
                                      _picked.remove(f.uid);
                                    } else {
                                      _picked.add(f.uid);
                                    }
                                  }),
                                ),
                                onTap: () => setState(() {
                                  if (on) {
                                    _picked.remove(f.uid);
                                  } else {
                                    _picked.add(f.uid);
                                  }
                                }),
                              );
                            },
                          ),
          ),
        ],
      ),
    );
  }
}
