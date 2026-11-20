import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/models.dart';
import '../core/util.dart';
import '../widgets/avatar.dart';
import 'chat_page.dart';
import 'me_page.dart';

class ConversationsPage extends StatefulWidget {
  const ConversationsPage({super.key});

  @override
  State<ConversationsPage> createState() => _ConversationsPageState();
}

class _ConversationsPageState extends State<ConversationsPage> {
  String _keyword = '';

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final list = app.convs.where((c) {
      if (_keyword.isEmpty) return true;
      final k = _keyword.toLowerCase();
      return c.title.toLowerCase().contains(k) || c.lastMsg.toLowerCase().contains(k);
    }).toList();

    return Scaffold(
      backgroundColor: const Color(0xFFEEF1F8),
      body: SafeArea(
        bottom: false,
        child: Column(
          children: [
            _header(app),
            _search(),
            Expanded(
              child: Container(
                clipBehavior: Clip.antiAlias,
                decoration: const BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
                ),
                child: RefreshIndicator(
                  onRefresh: () => app.loadConvs(),
                  child: list.isEmpty
                      ? ListView(
                          children: const [
                            SizedBox(height: 120),
                            Center(child: Text('暂无会话', style: TextStyle(color: Color(0xFF8A95AA)))),
                          ],
                        )
                      : ListView.builder(
                          padding: const EdgeInsets.only(top: 6, bottom: 18),
                          itemCount: list.length,
                          itemBuilder: (context, i) => _row(app, list[i]),
                        ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _header(AppState app) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 14, 12, 10),
      child: Row(
        children: [
          Expanded(
            child: GestureDetector(
              behavior: HitTestBehavior.opaque,
              onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const MePage())),
              child: Row(
                children: [
                  ClipOval(child: Avatar(name: app.myName, size: 46)),
                  const SizedBox(width: 11),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(app.myName.isEmpty ? 'NexIM' : app.myName,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            Container(
                              width: 6,
                              height: 6,
                              decoration: BoxDecoration(
                                shape: BoxShape.circle,
                                color: app.connected ? const Color(0xFF25B389) : const Color(0xFFC3C9D6),
                              ),
                            ),
                            const SizedBox(width: 5),
                            Text(app.connected ? '在线 · 4G' : '连接中…',
                                style: TextStyle(
                                    fontSize: 11.5,
                                    color: app.connected ? const Color(0xFF25B389) : const Color(0xFF9AA5B8))),
                          ],
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
          IconButton(
            tooltip: '更多',
            onPressed: _more,
            icon: const Icon(Icons.add, size: 30, color: Color(0xFF1B2333)),
          ),
        ],
      ),
    );
  }

  Widget _search() {
    return Container(
      height: 46,
      margin: const EdgeInsets.fromLTRB(14, 0, 14, 12),
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(24),
        boxShadow: const [BoxShadow(color: Color(0x141A2850), blurRadius: 14, offset: Offset(0, 4))],
      ),
      child: Row(
        children: [
          const Icon(Icons.search, size: 20, color: Color(0xFFADB3C1)),
          const SizedBox(width: 8),
          Expanded(
            child: TextField(
              textAlign: TextAlign.center,
              textAlignVertical: TextAlignVertical.center,
              onChanged: (v) => setState(() => _keyword = v.trim()),
              style: const TextStyle(fontSize: 15),
              decoration: const InputDecoration(
                isDense: true,
                border: InputBorder.none,
                hintText: '搜索',
                hintStyle: TextStyle(fontSize: 15, color: Color(0xFFADB3C1)),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _row(AppState app, Conv c) {
    return InkWell(
      onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => ChatPage(conv: c))),
      onLongPress: () => _actions(c),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(18, 11, 18, 11),
        child: Row(
          children: [
            SizedBox(
              width: 52,
              height: 52,
              child: Stack(
                clipBehavior: Clip.none,
                children: [
                  ClipOval(child: Avatar(name: c.title, url: c.peerAvatar, size: 52)),
                  if (c.unread > 0)
                    Positioned(
                      top: -4,
                      right: -4,
                      child: Container(
                        constraints: const BoxConstraints(minWidth: 18),
                        height: 18,
                        padding: const EdgeInsets.symmetric(horizontal: 5),
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(99),
                          boxShadow: const [
                            BoxShadow(color: Color(0x2E141E3C), blurRadius: 5, offset: Offset(0, 1))
                          ],
                        ),
                        child: Text(c.unread > 99 ? '99+' : '${c.unread}',
                            style: const TextStyle(
                                color: Color(0xFFF5405A), fontSize: 10.5, fontWeight: FontWeight.w700)),
                      ),
                    ),
                ],
              ),
            ),
            const SizedBox(width: 13),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Flexible(
                        child: Text(c.title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(fontSize: 16.5, fontWeight: FontWeight.w600)),
                      ),
                      if (c.pinned) ...[
                        const SizedBox(width: 6),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                          decoration: BoxDecoration(
                              color: const Color(0xFFFFF4E3), borderRadius: BorderRadius.circular(4)),
                          child: const Text('置顶',
                              style: TextStyle(
                                  fontSize: 9.5, color: Color(0xFFE39A2D), fontWeight: FontWeight.w600)),
                        ),
                      ],
                    ],
                  ),
                  const SizedBox(height: 5),
                  Text(
                    c.lastMsg.isEmpty ? '暂无消息' : (c.muted ? '[免打扰] ' : '') + c.lastMsg,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 14.5, color: Color(0xFFA2A9B8)),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            Padding(
              padding: const EdgeInsets.only(top: 3, bottom: 20),
              child: Text(timeText(c.lastMsgAt),
                  style: const TextStyle(fontSize: 12.5, color: Color(0xFFA2A9B8))),
            ),
          ],
        ),
      ),
    );
  }

  void _more() {
    final app = context.read<AppState>();
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.done_all),
              title: const Text('全部已读'),
              onTap: () {
                Navigator.pop(ctx);
                app.markAllRead();
              },
            ),
            ListTile(
              leading: const Icon(Icons.refresh),
              title: const Text('刷新会话'),
              onTap: () {
                Navigator.pop(ctx);
                app.loadConvs();
              },
            ),
          ],
        ),
      ),
    );
  }

  void _actions(Conv c) {
    final app = context.read<AppState>();
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: Icon(c.pinned ? Icons.push_pin : Icons.push_pin_outlined),
              title: Text(c.pinned ? '取消置顶' : '置顶会话'),
              onTap: () {
                Navigator.pop(ctx);
                app.togglePinned(c);
              },
            ),
            ListTile(
              leading: Icon(c.muted ? Icons.notifications_active_outlined : Icons.notifications_off_outlined),
              title: Text(c.muted ? '开启提醒' : '消息免打扰'),
              onTap: () {
                Navigator.pop(ctx);
                app.toggleMuted(c);
              },
            ),
            ListTile(
              leading: const Icon(Icons.delete_outline, color: Colors.red),
              title: const Text('删除会话', style: TextStyle(color: Colors.red)),
              onTap: () {
                Navigator.pop(ctx);
                app.deleteConv(c);
              },
            ),
          ],
        ),
      ),
    );
  }
}
