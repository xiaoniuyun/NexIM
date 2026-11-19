import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../core/models.dart';
import '../widgets/avatar.dart';
import 'chat_page.dart';

class ContactsPage extends StatefulWidget {
  const ContactsPage({super.key});

  @override
  State<ContactsPage> createState() => _ContactsPageState();
}

class _ContactsPageState extends State<ContactsPage> with SingleTickerProviderStateMixin {
  late final TabController _tab = TabController(length: 3, vsync: this);
  List<Contact> friends = [];
  List<Contact> members = [];
  List<GroupInfo> groups = [];
  List<Map<String, dynamic>> depts = [];
  int deptId = 0;
  bool loading = true;
  String error = '';

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _tab.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() {
      loading = true;
      error = '';
    });
    try {
      final results = await Future.wait([
        Api.get('/friends'),
        Api.get('/depts'),
        Api.get('/depts/0/members'),
        Api.get('/groups'),
      ]);
      List<T> parse<T>(dynamic data, T Function(Map<String, dynamic>) f) => (data is List)
          ? data.whereType<Map>().map((j) => f(Map<String, dynamic>.from(j))).toList()
          : <T>[];
      friends = parse(results[0], (j) => Contact.fromJson(j));
      depts = (results[1] is List)
          ? (results[1] as List).whereType<Map>().map((j) => Map<String, dynamic>.from(j)).toList()
          : [];
      members = parse(results[2], (j) => Contact.fromJson(j));
      groups = parse(results[3], (j) => GroupInfo.fromJson(j));
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> _pickDept(int id) async {
    setState(() => deptId = id);
    try {
      final data = await Api.get('/depts/$id/members');
      final list = (data is List)
          ? data.whereType<Map>().map((j) => Contact.fromJson(Map<String, dynamic>.from(j))).toList()
          : <Contact>[];
      if (mounted) setState(() => members = list);
    } catch (_) {}
  }

  void _openChat(String uid, String name) {
    final app = context.read<AppState>();
    final convId = app.ensureConv(uid, name);
    final conv = app.convs.firstWhere((c) => c.convId == convId);
    Navigator.of(context).push(MaterialPageRoute(builder: (_) => ChatPage(conv: conv)));
  }

  void _openGroup(GroupInfo g) {
    final app = context.read<AppState>();
    final convId = app.ensureGroupConv(g.groupId, g.name);
    final conv = app.convs.firstWhere((c) => c.convId == convId);
    Navigator.of(context).push(MaterialPageRoute(builder: (_) => ChatPage(conv: conv)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('通讯录'),
        actions: [
          IconButton(onPressed: _load, icon: const Icon(Icons.refresh)),
        ],
        bottom: TabBar(
          controller: _tab,
          tabs: const [
            Tab(text: '好友'),
            Tab(text: '组织架构'),
            Tab(text: '群聊'),
          ],
        ),
      ),
      body: loading
          ? const Center(child: CircularProgressIndicator())
          : error.isNotEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(error, style: const TextStyle(color: Colors.grey)),
                      const SizedBox(height: 10),
                      FilledButton(onPressed: _load, child: const Text('重试')),
                    ],
                  ),
                )
              : TabBarView(
                  controller: _tab,
                  children: [
                    _contactList(friends, '暂无好友'),
                    _deptView(),
                    ListView.separated(
                      itemCount: groups.length,
                      separatorBuilder: (_, __) => const Divider(height: 1, indent: 72),
                      itemBuilder: (context, i) {
                        final g = groups[i];
                        return ListTile(
                          leading: Avatar(name: g.name, size: 46),
                          title: Text(g.name, style: const TextStyle(fontWeight: FontWeight.w600)),
                          subtitle: Text('${g.memberCount} 位成员${g.notice.isEmpty ? '' : ' · ${g.notice}'}',
                              maxLines: 1, overflow: TextOverflow.ellipsis),
                          trailing: const Icon(Icons.chevron_right),
                          onTap: () => _openGroup(g),
                        );
                      },
                    ),
                  ],
                ),
    );
  }

  Widget _deptView() {
    return Column(
      children: [
        SizedBox(
          height: 48,
          child: ListView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            children: [
              _deptChip(0, '全部'),
              for (final d in depts) _deptChip((d['id'] as num?)?.toInt() ?? 0, '${d['name'] ?? ''}'),
            ],
          ),
        ),
        const Divider(height: 1),
        Expanded(child: _contactList(members, '暂无成员')),
      ],
    );
  }

  Widget _deptChip(int id, String label) {
    final on = deptId == id;
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: ChoiceChip(
        label: Text(label),
        selected: on,
        onSelected: (_) => _pickDept(id),
      ),
    );
  }

  Widget _contactList(List<Contact> list, String empty) {
    if (list.isEmpty) {
      return Center(child: Text(empty, style: const TextStyle(color: Colors.grey)));
    }
    return ListView.separated(
      itemCount: list.length,
      separatorBuilder: (_, __) => const Divider(height: 1, indent: 72),
      itemBuilder: (context, i) {
        final u = list[i];
        return ListTile(
          leading: Avatar(name: u.nickname, url: u.avatar, size: 46),
          title: Text(u.nickname, style: const TextStyle(fontWeight: FontWeight.w600)),
          subtitle: Text(
            [u.deptName, u.title, '@${u.username}'].where((s) => s.isNotEmpty).join(' · '),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          trailing: IconButton(
            tooltip: '发消息',
            onPressed: () => _openChat(u.uid, u.nickname),
            icon: const Icon(Icons.chat_bubble_outline),
          ),
          onTap: () => _openChat(u.uid, u.nickname),
        );
      },
    );
  }
}
