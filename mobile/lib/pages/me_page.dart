import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../core/util.dart';
import '../widgets/avatar.dart';

class MePage extends StatelessWidget {
  const MePage({super.key});

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final nickname = app.myName.isEmpty ? '未登录' : app.myName;
    return Scaffold(
      appBar: AppBar(title: const Text('我的')),
      body: ListView(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 20),
            child: Row(
              children: [
                Avatar(name: nickname, url: '${app.user?['avatar'] ?? ''}', size: 64),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(nickname, style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
                      const SizedBox(height: 4),
                      Text('@${app.user?['username'] ?? ''}',
                          style: const TextStyle(color: Colors.grey, fontSize: 13)),
                    ],
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: app.connected ? const Color(0xFFE8F6F1) : const Color(0xFFF1F3F6),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(app.connected ? '在线' : '离线',
                      style: TextStyle(
                          fontSize: 11,
                          color: app.connected ? const Color(0xFF3BA58C) : Colors.grey)),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          ListTile(
            leading: const Icon(Icons.dns_outlined),
            title: const Text('服务端地址'),
            subtitle: Text(app.apiBase, maxLines: 1, overflow: TextOverflow.ellipsis),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _editServer(context, app),
          ),
          ListTile(
            leading: const Icon(Icons.devices_outlined),
            title: const Text('登录设备'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _showDevices(context),
          ),
          ListTile(
            leading: const Icon(Icons.lock_outline),
            title: const Text('账号安全'),
            subtitle: const Text('密码修改请联系管理员'),
          ),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: const Text('关于 NexIM'),
            subtitle: const Text('版本 1.0.0 · Flutter 双端'),
          ),
          const SizedBox(height: 24),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: OutlinedButton(
              style: OutlinedButton.styleFrom(
                foregroundColor: Colors.red,
                side: const BorderSide(color: Color(0xFFE05B5B)),
                minimumSize: const Size.fromHeight(46),
              ),
              onPressed: () async {
                final ok = await showDialog<bool>(
                  context: context,
                  builder: (ctx) => AlertDialog(
                    title: const Text('退出登录'),
                    content: const Text('确定要退出当前账号吗？'),
                    actions: [
                      TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
                      TextButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('退出')),
                    ],
                  ),
                );
                if (ok == true) await app.logout();
              },
              child: const Text('退出登录'),
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _editServer(BuildContext context, AppState app) async {
    final ctrl = TextEditingController(text: app.apiBase);
    final saved = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('服务端地址'),
        content: TextField(
          controller: ctrl,
          keyboardType: TextInputType.url,
          decoration: const InputDecoration(hintText: 'http://192.168.1.10:8080/api/v1'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          TextButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('保存')),
        ],
      ),
    );
    if (saved == true) {
      await app.setServer(ctrl.text);
      app.connect();
    }
  }

  Future<void> _showDevices(BuildContext context) async {
    List<Map<String, dynamic>> list = [];
    String error = '';
    try {
      final data = await Api.get('/auth/devices');
      if (data is List) {
        list = data.whereType<Map>().map((j) => Map<String, dynamic>.from(j)).toList();
      }
    } catch (e) {
      error = '$e';
    }
    if (!context.mounted) return;
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('登录设备'),
        content: SizedBox(
          width: 320,
          child: error.isNotEmpty
              ? Text(error)
              : list.isEmpty
                  ? const Text('暂无其他设备')
                  : Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        for (final d in list)
                          ListTile(
                            dense: true,
                            contentPadding: EdgeInsets.zero,
                            leading: Icon(d['platform'] == 'iOS' || d['platform'] == 'Android'
                                ? Icons.phone_iphone
                                : Icons.computer),
                            title: Text('${d['deviceName'] ?? d['deviceId'] ?? ''}'),
                            subtitle: Text(
                                '${d['platform'] ?? ''} · ${fullTime((d['lastActive'] as num?)?.toInt() ?? 0)}'),
                          ),
                      ],
                    ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('关闭')),
        ],
      ),
    );
  }
}
