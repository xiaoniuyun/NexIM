import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../app/app_state.dart';
import '../widgets/avatar.dart';
import 'scan_page.dart';

/// 我的名片：展示本人头像与联系方式，可一键复制，便于线下交换信息
class MyCardPage extends StatefulWidget {
  const MyCardPage({super.key});

  @override
  State<MyCardPage> createState() => _MyCardPageState();
}

class _MyCardPageState extends State<MyCardPage> {
  @override
  void initState() {
    super.initState();
    // 进入名片时拉取最新资料（手机号/邮箱/组织由服务端维护）
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<AppState>().refreshProfile();
    });
  }

  void _copy(String label, String value) {
    if (value.isEmpty) {
      _toast('$label 暂未填写');
      return;
    }
    Clipboard.setData(ClipboardData(text: value));
    _toast('$label 已复制');
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(msg), duration: const Duration(seconds: 1)));
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final u = app.user ?? {};
    final nickname = '${u['nickname'] ?? u['username'] ?? ''}';
    final avatar = '${u['avatar'] ?? ''}';
    final username = '${u['username'] ?? ''}';
    final uid = '${u['uid'] ?? ''}';
    final phone = '${u['phone'] ?? ''}';
    final email = '${u['email'] ?? ''}';
    final org = '${u['orgName'] ?? ''}'.isNotEmpty ? '${u['orgName']}' : '${u['deptName'] ?? ''}';
    final title = '${u['title'] ?? ''}';
    final external = u['isExternal'] == true;

    return Scaffold(
      backgroundColor: const Color(0xFFF5F7FB),
      appBar: AppBar(
        title: const Text('我的名片'),
        actions: [
          IconButton(
            tooltip: '扫一扫',
            onPressed: () => Navigator.push(context, MaterialPageRoute(builder: (_) => const ScanPage())),
            icon: const Icon(Icons.qr_code_scanner_rounded),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(20, 22, 20, 20),
            decoration: BoxDecoration(
              gradient: const LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [Color(0xFF1B7FFF), Color(0xFF5A9BFF)],
              ),
              borderRadius: BorderRadius.circular(18),
              boxShadow: const [
                BoxShadow(color: Color(0x331B7FFF), blurRadius: 18, offset: Offset(0, 8)),
              ],
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Avatar(name: nickname, url: avatar, size: 64),
                    const SizedBox(width: 14),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Flexible(
                                child: Text(
                                  nickname.isEmpty ? '未设置昵称' : nickname,
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: const TextStyle(
                                      color: Colors.white, fontSize: 21, fontWeight: FontWeight.w700),
                                ),
                              ),
                              if (external) ...[
                                const SizedBox(width: 6),
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                  decoration: BoxDecoration(
                                    color: const Color(0x38FFFFFF),
                                    borderRadius: BorderRadius.circular(6),
                                  ),
                                  child: const Text('外部联系人',
                                      style: TextStyle(color: Colors.white, fontSize: 10.5)),
                                ),
                              ],
                            ],
                          ),
                          const SizedBox(height: 5),
                          Text(
                            title.isEmpty ? 'NexIM 用户' : title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(color: Color(0xD9FFFFFF), fontSize: 13),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 20),
                Container(height: 1, color: const Color(0x33FFFFFF)),
                const SizedBox(height: 14),
                _CardLine(icon: Icons.alternate_email, label: '@$username'),
                if (org.isNotEmpty) _CardLine(icon: Icons.business_outlined, label: org),
                _CardLine(icon: Icons.badge_outlined, label: 'UID $uid'),
              ],
            ),
          ),
          const SizedBox(height: 18),
          // 我的二维码：供他人扫一扫添加联系人
          Container(
            width: double.infinity,
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 18),
            decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14)),
            child: Column(
              children: [
                Row(
                  children: [
                    const Icon(Icons.qr_code_2_rounded, size: 18, color: Color(0xFF5B6880)),
                    const SizedBox(width: 6),
                    const Text('我的二维码',
                        style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: Color(0xFF1B2333))),
                    const Spacer(),
                    TextButton.icon(
                      onPressed: () =>
                          Navigator.push(context, MaterialPageRoute(builder: (_) => const ScanPage())),
                      icon: const Icon(Icons.qr_code_scanner_rounded, size: 16),
                      label: const Text('扫一扫', style: TextStyle(fontSize: 12.5)),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (uid.isEmpty)
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 30),
                    child: Text('资料加载中…', style: TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA))),
                  )
                else
                  Container(
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(10),
                      border: Border.all(color: const Color(0xFFEDF0F6)),
                    ),
                    child: QrImageView(
                      data: 'NEXIM:$uid',
                      version: QrVersions.auto,
                      size: 172,
                      gapless: true,
                      eyeStyle: const QrEyeStyle(
                        eyeShape: QrEyeShape.square,
                        color: Color(0xFF1B2333),
                      ),
                      dataModuleStyle: const QrDataModuleStyle(
                        dataModuleShape: QrDataModuleShape.square,
                        color: Color(0xFF1B2333),
                      ),
                    ),
                  ),
                const SizedBox(height: 10),
                const Text('让对方用 NexIM「扫一扫」即可加我为联系人',
                    style: TextStyle(fontSize: 12, color: Color(0xFF8A95AA))),
              ],
            ),
          ),
          const SizedBox(height: 18),
          Container(
            decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(14)),
            child: Column(
              children: [
                _InfoTile(
                  icon: Icons.phone_outlined,
                  label: '手机号',
                  value: phone.isEmpty ? '未绑定' : phone,
                  onCopy: () => _copy('手机号', phone),
                ),
                const Divider(height: 1, indent: 56, color: Color(0xFFF2F4F9)),
                _InfoTile(
                  icon: Icons.mail_outline,
                  label: '邮箱',
                  value: email.isEmpty ? '未填写' : email,
                  onCopy: () => _copy('邮箱', email),
                ),
                const Divider(height: 1, indent: 56, color: Color(0xFFF2F4F9)),
                _InfoTile(
                  icon: Icons.apartment_outlined,
                  label: '所在组织',
                  value: org.isEmpty ? '未加入组织' : org,
                ),
                const Divider(height: 1, indent: 56, color: Color(0xFFF2F4F9)),
                _InfoTile(icon: Icons.work_outline, label: '职位', value: title.isEmpty ? '未填写' : title),
              ],
            ),
          ),
          const SizedBox(height: 14),
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 4),
            child: Text('手机号与所属组织由管理员维护，邮箱可在「编辑资料」中修改',
                style: TextStyle(fontSize: 12, color: Color(0xFF8A95AA))),
          ),
        ],
      ),
    );
  }
}

class _CardLine extends StatelessWidget {
  final IconData icon;
  final String label;
  const _CardLine({required this.icon, required this.label});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        children: [
          Icon(icon, size: 15, color: const Color(0xCCFFFFFF)),
          const SizedBox(width: 8),
          Expanded(
            child: Text(label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(color: Color(0xE6FFFFFF), fontSize: 13)),
          ),
        ],
      ),
    );
  }
}

class _InfoTile extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final VoidCallback? onCopy;
  const _InfoTile({required this.icon, required this.label, required this.value, this.onCopy});

  @override
  Widget build(BuildContext context) {
    return ListTile(
      leading: Icon(icon, color: const Color(0xFF5B6880)),
      title: Text(label, style: const TextStyle(fontSize: 14, color: Color(0xFF1B2333))),
      subtitle: Text(value, style: const TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA))),
      trailing: onCopy == null
          ? null
          : IconButton(
              tooltip: '复制',
              onPressed: onCopy,
              icon: const Icon(Icons.copy_rounded, size: 18, color: Color(0xFF8A95AA)),
            ),
    );
  }
}
