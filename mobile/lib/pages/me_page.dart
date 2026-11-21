import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../core/util.dart';
import '../widgets/avatar.dart';
import 'change_password_page.dart';
import 'edit_profile_page.dart';
import 'my_card_page.dart';
import 'scan_page.dart';

class MePage extends StatefulWidget {
  const MePage({super.key});

  @override
  State<MePage> createState() => _MePageState();
}

class _MePageState extends State<MePage> {
  @override
  void initState() {
    super.initState();
    // 进入「我的」时刷新资料，保证手机号/邮箱/所在组织为最新
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().refreshProfile();
    });
  }

  void _openCard() {
    Navigator.push(context, MaterialPageRoute(builder: (_) => const MyCardPage()));
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final u = app.user ?? {};
    final nickname = app.myName.isEmpty ? '未登录' : app.myName;
    final phone = '${u['phone'] ?? ''}';
    final email = '${u['email'] ?? ''}';
    final org = '${u['orgName'] ?? ''}'.isNotEmpty ? '${u['orgName']}' : '${u['deptName'] ?? ''}';
    final title = '${u['title'] ?? ''}';
    final external = u['isExternal'] == true;

    return Scaffold(
      backgroundColor: const Color(0xFFF5F7FB),
      appBar: AppBar(
        title: const Text('我的'),
        actions: [
          IconButton(
            tooltip: '扫一扫',
            onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const ScanPage())),
            icon: const Icon(Icons.qr_code_scanner_rounded),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 10, 16, 28),
        children: [
          // 名片与资料合并：一张卡片承载身份、联系方式与快捷操作
          GestureDetector(onTap: _openCard, child: _profileCard(nickname, phone, email, org, title, external, u)),
          const SizedBox(height: 16),
          _group([
            ListTile(
              leading: const Icon(Icons.qr_code_scanner_rounded, color: Color(0xFF2F7DF6)),
              title: const Text('扫一扫'),
              subtitle: const Text('扫描二维码，添加联系人或查看名片'),
              trailing: const Icon(Icons.chevron_right, color: Color(0xFFC3CAD9)),
              onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const ScanPage())),
            ),
            ListTile(
              leading: const Icon(Icons.lock_outline, color: Color(0xFF5B6880)),
              title: const Text('修改密码'),
              subtitle: const Text('验证原密码后设置新密码'),
              trailing: const Icon(Icons.chevron_right, color: Color(0xFFC3CAD9)),
              onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const ChangePasswordPage())),
            ),
          ]),
          const SizedBox(height: 12),
          _group([
            ListTile(
              leading: const Icon(Icons.dns_outlined, color: Color(0xFF5B6880)),
              title: const Text('服务端地址'),
              subtitle: Text(app.apiBase, maxLines: 1, overflow: TextOverflow.ellipsis),
              trailing: const Icon(Icons.chevron_right, color: Color(0xFFC3CAD9)),
              onTap: () => _editServer(context, app),
            ),
            ListTile(
              leading: const Icon(Icons.devices_outlined, color: Color(0xFF5B6880)),
              title: const Text('登录设备'),
              trailing: const Icon(Icons.chevron_right, color: Color(0xFFC3CAD9)),
              onTap: () => _showDevices(context),
            ),
            const ListTile(
              leading: Icon(Icons.info_outline, color: Color(0xFF5B6880)),
              title: Text('关于 NexIM'),
              subtitle: Text('版本 1.0.0 · Flutter 双端'),
            ),
          ]),
          const SizedBox(height: 22),
          OutlinedButton(
            style: OutlinedButton.styleFrom(
              foregroundColor: const Color(0xFFE05B5B),
              side: const BorderSide(color: Color(0xFFF0C9C9)),
              backgroundColor: Colors.white,
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
              if (ok == true) {
                await app.logout();
                // 退出后关闭所有已打开页面，回到登录页
                if (context.mounted) Navigator.of(context).popUntil((r) => r.isFirst);
              }
            },
            child: const Text('退出登录'),
          ),
        ],
      ),
    );
  }

  Widget _profileCard(String nickname, String phone, String email, String org, String title, bool external,
      Map<String, dynamic> u) {
    final avatar = '${u['avatar'] ?? ''}';
    final sub = org.isNotEmpty ? org : '@${u['username'] ?? ''}';
    return Container(
      padding: const EdgeInsets.fromLTRB(18, 18, 18, 14),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [Color(0xFF1B7FFF), Color(0xFF5A9BFF)],
        ),
        borderRadius: BorderRadius.circular(18),
        boxShadow: const [BoxShadow(color: Color(0x2A1B7FFF), blurRadius: 16, offset: Offset(0, 8))],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Avatar(name: nickname, url: avatar, size: 58),
              const SizedBox(width: 13),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Flexible(
                          child: Text(nickname,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                  color: Colors.white, fontSize: 19, fontWeight: FontWeight.w700)),
                        ),
                        if (external) ...[
                          const SizedBox(width: 6),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                                color: const Color(0x38FFFFFF), borderRadius: BorderRadius.circular(5)),
                            child: const Text('外部联系人', style: TextStyle(color: Colors.white, fontSize: 10)),
                          ),
                        ],
                      ],
                    ),
                    const SizedBox(height: 4),
                    Text(sub,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(color: Color(0xD9FFFFFF), fontSize: 12.5)),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right, color: Color(0xB3FFFFFF)),
            ],
          ),
          const SizedBox(height: 16),
          Container(height: 1, color: const Color(0x33FFFFFF)),
          const SizedBox(height: 12),
          _line(Icons.phone_outlined, phone.isEmpty ? '手机号未绑定' : phone, phone),
          _line(Icons.mail_outline, email.isEmpty ? '邮箱未填写' : email, email),
          _line(Icons.apartment_outlined, org.isEmpty ? '未加入组织' : org, ''),
          if (title.isNotEmpty) _line(Icons.work_outline, title, ''),
          const SizedBox(height: 8),
          Row(
            children: [
              Expanded(
                child: _cardAction(
                  icon: Icons.edit_outlined,
                  label: '编辑资料',
                  onTap: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const EditProfilePage())),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: _cardAction(
                  icon: Icons.qr_code_2_rounded,
                  label: '我的二维码',
                  onTap: _openCard,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _line(IconData icon, String text, String copyValue) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 9),
      child: Row(
        children: [
          Icon(icon, size: 15, color: const Color(0xCCFFFFFF)),
          const SizedBox(width: 8),
          Expanded(
            child: Text(text,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(color: Color(0xE6FFFFFF), fontSize: 12.5)),
          ),
          if (copyValue.isNotEmpty)
            GestureDetector(
              onTap: () {
                Clipboard.setData(ClipboardData(text: copyValue));
                ScaffoldMessenger.of(context)
                  ..hideCurrentSnackBar()
                  ..showSnackBar(const SnackBar(content: Text('已复制')));
              },
              child: const Padding(
                padding: EdgeInsets.only(left: 6),
                child: Icon(Icons.copy_rounded, size: 14, color: Color(0xB3FFFFFF)),
              ),
            ),
        ],
      ),
    );
  }

  Widget _cardAction({required IconData icon, required String label, required VoidCallback onTap}) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        height: 36,
        decoration: BoxDecoration(color: const Color(0x2EFFFFFF), borderRadius: BorderRadius.circular(9)),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, size: 15, color: Colors.white),
            const SizedBox(width: 5),
            Text(label, style: const TextStyle(color: Colors.white, fontSize: 12.5)),
          ],
        ),
      ),
    );
  }

  Widget _group(List<Widget> children) {
    return Container(
      decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14)),
      child: Column(
        children: [
          for (var i = 0; i < children.length; i++) ...[
            if (i > 0) const Divider(height: 1, indent: 56, color: Color(0xFFF2F4F9)),
            children[i],
          ],
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
