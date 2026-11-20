import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../app/call_state.dart';
import '../core/api.dart';
import '../widgets/avatar.dart';
import 'chat_page.dart';
import 'contacts_page.dart';
import 'conversations_page.dart';

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int _index = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().loadConvs();
    });
  }

  @override
  Widget build(BuildContext context) {
    final unread = context.watch<AppState>().unreadTotal;
    final inCall = context.watch<CallState>().busy;
    return Scaffold(
      body: IndexedStack(
        index: _index,
        children: const [ConversationsPage(), _ChannelsView(), ContactsPage(), _MomentsView()],
      ),
      bottomNavigationBar: Container(
        decoration: const BoxDecoration(
          color: Colors.white,
          border: Border(top: BorderSide(color: Color(0xFFEDEFF4), width: 0.6)),
        ),
        child: SafeArea(
          top: false,
          child: SizedBox(
            height: 56,
            child: Row(
              children: [
                _nav(0, Icons.chat_bubble, Icons.chat_bubble, '消息',
                    badge: unread > 0 && !inCall ? (unread > 99 ? '99+' : '$unread') : null),
                _nav(1, Icons.tag, Icons.tag, '频道'),
                _nav(2, Icons.person_outline, Icons.person, '联系人'),
                _nav(3, Icons.autorenew, Icons.autorenew, '动态'),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _nav(int index, IconData icon, IconData activeIcon, String label, {String? badge}) {
    final on = _index == index;
    final color = on ? const Color(0xFF1B7FFF) : const Color(0xFF2B2F38);
    return Expanded(
      child: InkWell(
        onTap: () => setState(() => _index = index),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            SizedBox(
              width: 34,
              height: 26,
              child: Stack(
                clipBehavior: Clip.none,
                alignment: Alignment.center,
                children: [
                  Icon(on ? activeIcon : icon, size: 24, color: color),
                  if (badge != null)
                    Positioned(
                      top: -5,
                      right: -6,
                      child: Container(
                        constraints: const BoxConstraints(minWidth: 15),
                        height: 15,
                        padding: const EdgeInsets.symmetric(horizontal: 3),
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: const Color(0xFFF5405A),
                          borderRadius: BorderRadius.circular(99),
                          border: Border.all(color: Colors.white, width: 1.5),
                        ),
                        child: Text(badge,
                            style: const TextStyle(color: Colors.white, fontSize: 9, fontWeight: FontWeight.w700)),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 3),
            Text(label,
                style: TextStyle(
                    fontSize: 10.5, color: color, fontWeight: on ? FontWeight.w700 : FontWeight.w400)),
          ],
        ),
      ),
    );
  }
}

/// 频道：聚合群聊会话
class _ChannelsView extends StatelessWidget {
  const _ChannelsView();

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final groups = app.convs.where((c) => c.isGroup).toList();
    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(18, 14, 18, 10),
              child: Row(
                children: [
                  const Text('频道', style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
                  const Spacer(),
                  Text('${groups.length} 个群聊',
                      style: const TextStyle(fontSize: 12, color: Color(0xFF9AA5B8))),
                ],
              ),
            ),
            Expanded(
              child: groups.isEmpty
                  ? const Center(child: Text('暂无频道', style: TextStyle(color: Color(0xFF8A95AA))))
                  : ListView.separated(
                      itemCount: groups.length,
                      separatorBuilder: (_, __) => const Divider(
                          height: 1, thickness: 0.6, indent: 76, color: Color(0xFFF2F4F9)),
                      itemBuilder: (context, i) {
                        final c = groups[i];
                        return ListTile(
                          contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 4),
                          leading: ClipOval(child: Avatar(name: c.title, url: c.peerAvatar, size: 48)),
                          title: Text(c.title,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                          subtitle: Text(c.lastMsg.isEmpty ? '暂无消息' : c.lastMsg,
                              maxLines: 1, overflow: TextOverflow.ellipsis),
                          onTap: () => Navigator.of(context)
                              .push(MaterialPageRoute(builder: (_) => ChatPage(conv: c))),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

/// 动态：企业公告
class _MomentsView extends StatefulWidget {
  const _MomentsView();

  @override
  State<_MomentsView> createState() => _MomentsViewState();
}

class _MomentsViewState extends State<_MomentsView> {
  List<Map<String, dynamic>> notices = [];
  bool loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await Api.get('/notices');
      notices = (data is List)
          ? data.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList()
          : [];
    } catch (_) {}
    if (mounted) setState(() => loading = false);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.white,
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            const Padding(
              padding: EdgeInsets.fromLTRB(18, 14, 18, 10),
              child: Row(children: [
                Text('动态', style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
              ]),
            ),
            Expanded(
              child: RefreshIndicator(
                onRefresh: _load,
                child: loading
                    ? const Center(child: CircularProgressIndicator())
                    : ListView.separated(
                        padding: const EdgeInsets.fromLTRB(16, 4, 16, 20),
                        itemCount: notices.isEmpty ? 1 : notices.length,
                        separatorBuilder: (_, __) => const SizedBox(height: 10),
                        itemBuilder: (context, i) {
                          if (notices.isEmpty) {
                            return const Padding(
                              padding: EdgeInsets.only(top: 120),
                              child: Center(
                                  child: Text('暂无动态', style: TextStyle(color: Color(0xFF8A95AA)))),
                            );
                          }
                          final n = notices[i];
                          return Container(
                            padding: const EdgeInsets.all(14),
                            decoration: BoxDecoration(
                              color: const Color(0xFFF6F8FC),
                              borderRadius: BorderRadius.circular(14),
                            ),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text('${n['title'] ?? ''}',
                                    style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
                                const SizedBox(height: 6),
                                Text('${n['content'] ?? ''}',
                                    style: const TextStyle(
                                        fontSize: 13, color: Color(0xFF6B7688), height: 1.5)),
                              ],
                            ),
                          );
                        },
                      ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

