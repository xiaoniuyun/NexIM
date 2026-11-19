import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/models.dart';
import '../core/util.dart';
import '../widgets/avatar.dart';
import 'chat_page.dart';

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
      appBar: AppBar(
        title: Row(
          children: [
            const Text('NexIM', style: TextStyle(fontWeight: FontWeight.w700)),
            const SizedBox(width: 8),
            Container(
              width: 7,
              height: 7,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: app.connected ? const Color(0xFF3BA58C) : Colors.grey,
              ),
            ),
            const SizedBox(width: 4),
            Text(app.connected ? '在线' : '连接中',
                style: const TextStyle(fontSize: 11, color: Colors.grey, fontWeight: FontWeight.normal)),
          ],
        ),
        actions: [
          IconButton(
            tooltip: '全部已读',
            onPressed: () => app.markAllRead(),
            icon: const Icon(Icons.done_all),
          ),
          IconButton(
            tooltip: '刷新',
            onPressed: () => app.loadConvs(),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
            child: TextField(
              onChanged: (v) => setState(() => _keyword = v.trim()),
              decoration: InputDecoration(
                hintText: '搜索会话',
                isDense: true,
                prefixIcon: const Icon(Icons.search, size: 20),
                filled: true,
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(10),
                  borderSide: BorderSide.none,
                ),
              ),
            ),
          ),
          Expanded(
            child: RefreshIndicator(
              onRefresh: () => app.loadConvs(),
              child: list.isEmpty
                  ? ListView(
                      children: const [
                        SizedBox(height: 120),
                        Center(child: Text('暂无会话', style: TextStyle(color: Colors.grey))),
                      ],
                    )
                  : ListView.separated(
                      itemCount: list.length,
                      separatorBuilder: (_, __) => const Divider(height: 1, indent: 74),
                      itemBuilder: (context, i) {
                        final c = list[i];
                        return ListTile(
                          onTap: () => Navigator.of(context).push(
                            MaterialPageRoute(builder: (_) => ChatPage(conv: c)),
                          ),
                          onLongPress: () => _actions(c),
                          leading: Badge(
                            isLabelVisible: c.unread > 0,
                            label: Text(c.unread > 99 ? '99+' : '${c.unread}'),
                            child: Avatar(name: c.title, url: c.peerAvatar, size: 48),
                          ),
                          title: Row(
                            children: [
                              Flexible(
                                child: Text(c.title,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: const TextStyle(fontWeight: FontWeight.w600)),
                              ),
                              if (c.isGroup) ...[
                                const SizedBox(width: 6),
                                const Icon(Icons.group, size: 13, color: Colors.grey),
                              ],
                              if (c.pinned) ...[
                                const SizedBox(width: 4),
                                const Icon(Icons.push_pin, size: 13, color: Colors.grey),
                              ],
                            ],
                          ),
                          subtitle: Text(c.lastMsg.isEmpty ? '暂无消息' : c.lastMsg,
                              maxLines: 1, overflow: TextOverflow.ellipsis),
                          trailing: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            crossAxisAlignment: CrossAxisAlignment.end,
                            children: [
                              Text(timeText(c.lastMsgAt),
                                  style: const TextStyle(fontSize: 11, color: Colors.grey)),
                              const SizedBox(height: 6),
                              if (c.muted)
                                const Icon(Icons.notifications_off_outlined, size: 15, color: Colors.grey)
                              else if (c.unread > 0)
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                                  decoration: BoxDecoration(
                                    color: const Color(0xFFE05B5B),
                                    borderRadius: BorderRadius.circular(9),
                                  ),
                                  child: Text(c.unread > 99 ? '99+' : '${c.unread}',
                                      style: const TextStyle(color: Colors.white, fontSize: 10)),
                                )
                              else
                                const SizedBox(height: 15),
                            ],
                          ),
                        );
                      },
                    ),
            ),
          ),
        ],
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
