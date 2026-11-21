import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../core/models.dart';
import '../widgets/avatar.dart';
import 'add_contact_page.dart';
import 'chat_page.dart';
import 'create_group_page.dart';

class ContactsPage extends StatefulWidget {
  const ContactsPage({super.key});

  @override
  State<ContactsPage> createState() => _ContactsPageState();
}

class _ContactsPageState extends State<ContactsPage> with SingleTickerProviderStateMixin {
  late final bool external;
  late final TabController _tab;
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
    // 外部联系人默认无法查看组织架构：隐藏该页签且不请求部门数据
    external = context.read<AppState>().user?['isExternal'] == true;
    _tab = TabController(length: external ? 2 : 3, vsync: this);
    // 页签切换后右上角按钮行为随之变化，需要刷新
    _tab.addListener(() {
      if (_tab.indexIsChanging) return;
      if (mounted) setState(() {});
    });
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
        if (!external) Api.get('/depts'),
        if (!external) Api.get('/depts/0/members'),
        Api.get('/groups'),
      ]);
      List<T> parse<T>(dynamic data, T Function(Map<String, dynamic>) f) => (data is List)
          ? data.whereType<Map>().map((j) => f(Map<String, dynamic>.from(j))).toList()
          : <T>[];
      friends = parse(results[0], (j) => Contact.fromJson(j));
      if (external) {
        depts = [];
        members = [];
      } else {
        depts = (results[1] is List)
            ? (results[1] as List).whereType<Map>().map((j) => Map<String, dynamic>.from(j)).toList()
            : [];
        members = parse(results[2], (j) => Contact.fromJson(j));
      }
      groups = parse(results[external ? 1 : 3], (j) => GroupInfo.fromJson(j));
      // 外部联系人不可见公司全员群（内部组织架构群）
      if (external) groups.removeWhere((g) => g.groupId == 'g_company');
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> _pickDept(int id) async {
    if (external) return;
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

  int get _groupTabIndex => external ? 1 : 2;

  /// 右上角「+」：外部联系人页 -> 添加联系人；群聊页 -> 发起群聊；组织架构页不显示
  VoidCallback? get _plusAction {
    final i = _tab.index;
    if (i == 0) return _addContact;
    if (i == _groupTabIndex) return _createGroup;
    return null;
  }

  Future<void> _addContact() async {
    await Navigator.of(context).push(MaterialPageRoute(builder: (_) => const AddContactPage()));
    if (mounted) await _load();
  }

  Future<void> _createGroup() async {
    final g = await Navigator.of(context).push<Map<String, dynamic>>(
      MaterialPageRoute(builder: (_) => const CreateGroupPage()),
    );
    if (g == null || !mounted) return;
    final groupId = '${g['groupId'] ?? ''}';
    if (groupId.isEmpty) return;
    final name = '${g['name'] ?? '群聊'}';
    final app = context.read<AppState>();
    await app.loadConvs();
    final convId = app.ensureGroupConv(groupId, name);
    final conv = app.convs.firstWhere((c) => c.convId == convId);
    if (!mounted) return;
    await Navigator.of(context).push(MaterialPageRoute(builder: (_) => ChatPage(conv: conv)));
    if (mounted) await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      appBar: AppBar(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.white,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('通讯录', style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700)),
            const SizedBox(height: 2),
            Text(external ? '外部联系人（不含组织架构）' : '企业内部成员与外部联系人',
                style: const TextStyle(fontSize: 11.5, color: Color(0xFF8A95AA), fontWeight: FontWeight.w400)),
          ],
        ),
        actions: [
          if (_plusAction != null)
            IconButton(
              tooltip: _tab.index == 0 ? '添加外部联系人' : '发起群聊',
              onPressed: _plusAction,
              icon: const Icon(Icons.add, size: 26),
            ),
        ],
        bottom: TabBar(
          controller: _tab,
          labelColor: const Color(0xFF1B2333),
          unselectedLabelColor: const Color(0xFF8A95AA),
          labelStyle: const TextStyle(fontSize: 13.5, fontWeight: FontWeight.w700),
          unselectedLabelStyle: const TextStyle(fontSize: 13.5),
          indicatorColor: const Color(0xFF1B7FFF),
          indicatorSize: TabBarIndicatorSize.label,
          tabs: [
            const Tab(text: '外部联系人'),
            if (!external) const Tab(text: '组织架构'),
            const Tab(text: '群聊'),
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
                    _contactList(friends, '暂无外部联系人'),
                    if (!external) _deptView(),
                    ListView.separated(
                      padding: EdgeInsets.zero,
                      itemCount: groups.length,
                      separatorBuilder: (_, __) => const Divider(
                          height: 1, thickness: 0.6, indent: 74, color: Color(0xFFF2F4F9)),
                      itemBuilder: (context, i) {
                        final g = groups[i];
                        return ListTile(
                          onTap: () => _openGroup(g),
                          leading: Avatar(name: g.name, size: 46),
                          title: Text(g.name, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                          subtitle: Text('${g.memberCount} 位成员${g.notice.isEmpty ? '' : ' · ${g.notice}'}',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(fontSize: 13, color: Color(0xFF8A95AA))),
                          trailing: const Icon(Icons.chevron_right, color: Color(0xFFC3CAD9)),
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
      child: GestureDetector(
        onTap: () => _pickDept(id),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 13, vertical: 6),
          decoration: BoxDecoration(
            color: on ? const Color(0xFF1B7FFF) : const Color(0xFFF3F5F9),
            borderRadius: BorderRadius.circular(99),
          ),
          child: Text(label,
              style: TextStyle(
                  fontSize: 12.5,
                  color: on ? Colors.white : const Color(0xFF4A556B),
                  fontWeight: on ? FontWeight.w600 : FontWeight.w400)),
        ),
      ),
    );
  }

  Widget _contactList(List<Contact> list, String empty) {
    if (list.isEmpty) {
      return Center(child: Text(empty, style: const TextStyle(color: Color(0xFF8A95AA))));
    }
    return ListView.separated(
      padding: EdgeInsets.zero,
      itemCount: list.length,
      separatorBuilder: (_, __) =>
          const Divider(height: 1, thickness: 0.6, indent: 74, color: Color(0xFFF2F4F9)),
      itemBuilder: (context, i) {
        final u = list[i];
        return ListTile(
          onTap: () => _openChat(u.uid, u.nickname),
          leading: Avatar(name: u.nickname, url: u.avatar, size: 46),
          title: Row(children: [
            Flexible(
                child: Text(u.nickname,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600))),
            if (u.external) ...[
              const SizedBox(width: 6),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                decoration: BoxDecoration(color: const Color(0xFFFFF3EC), borderRadius: BorderRadius.circular(4)),
                child: const Text('外部', style: TextStyle(fontSize: 10, color: Color(0xFFFF7A45), fontWeight: FontWeight.w600)),
              ),
            ],
          ]),
          subtitle: Text(
            [u.external ? '企业外部用户' : '', u.orgName.isNotEmpty ? u.orgName : u.deptName, u.title, '@${u.username}']
                .where((s) => s.isNotEmpty)
                .join(' · '),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(fontSize: 13, color: Color(0xFF8A95AA)),
          ),
          trailing: const Icon(Icons.chevron_right, color: Color(0xFFC3CAD9)),
        );
      },
    );
  }
}
