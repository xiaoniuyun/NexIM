import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../core/models.dart';
import '../widgets/avatar.dart';

/// 扫一扫：识别 NexIM 名片二维码（内容约定 `NEXIM:<uid>`），展示用户资料并支持添加联系人
class ScanPage extends StatefulWidget {
  const ScanPage({super.key});

  @override
  State<ScanPage> createState() => _ScanPageState();
}

class _ScanPageState extends State<ScanPage> {
  final MobileScannerController _controller = MobileScannerController(
    detectionSpeed: DetectionSpeed.noDuplicates,
    formats: const [BarcodeFormat.qrCode],
    facing: CameraFacing.back,
  );
  bool _handling = false;
  bool _torch = false;
  String _hint = '将二维码放入框内，即可自动识别';

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  /// 从二维码内容解析 uid：支持 `NEXIM:<uid>`、`nexim://u/<uid>` 与裸 uid
  String? _parseUid(String raw) {
    final text = raw.trim();
    if (text.isEmpty) return null;
    if (text.toUpperCase().startsWith('NEXIM:')) {
      final v = text.substring(6).trim();
      return RegExp(r'^u_[A-Za-z0-9]+$').hasMatch(v) ? v : null;
    }
    final m = RegExp(r'u_[A-Za-z0-9]+').firstMatch(text);
    return m?.group(0);
  }

  Future<void> _onDetect(BarcodeCapture capture) async {
    if (_handling) return;
    final raw = capture.barcodes.map((b) => b.rawValue).firstWhere((v) => v != null && v.isNotEmpty, orElse: () => null);
    if (raw == null) return;
    final targetUid = _parseUid(raw);
    if (targetUid == null) {
      setState(() => _hint = '未识别的二维码，请扫描 NexIM 名片');
      return;
    }
    if (targetUid == context.read<AppState>().myUid) {
      setState(() => _hint = '这是你自己的名片二维码');
      return;
    }
    _handling = true;
    await _controller.stop();
    if (!mounted) return;
    try {
      final results = await Future.wait([
        Api.get('/users/$targetUid/profile'),
        Api.get('/friends'),
      ]);
      final data = results[0];
      if (data == null) {
        _fail('未找到该用户（可能不属于当前企业/服务端）');
        return;
      }
      final user = Contact.fromJson(Map<String, dynamic>.from(data as Map));
      final friends = (results[1] is List)
          ? (results[1] as List).whereType<Map>().map((j) => Contact.fromJson(Map<String, dynamic>.from(j))).toList()
          : <Contact>[];
      final isFriend = friends.any((f) => f.uid == user.uid);
      if (!mounted) return;
      final action = await showModalBottomSheet<String>(
        context: context,
        backgroundColor: Colors.white,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(18))),
        builder: (_) => _ScanResultSheet(user: user, isFriend: isFriend),
      );
      if (action == 'add') {
        await _apply(user);
      } else {
        _resume();
      }
    } catch (e) {
      _fail(e is ApiException ? e.message : '识别失败，请重试');
    }
  }

  void _fail(String msg) {
    if (!mounted) return;
    setState(() => _hint = msg);
    _resume();
  }

  Future<void> _resume() async {
    _handling = false;
    try {
      await _controller.start();
    } catch (_) {}
    if (mounted) setState(() {});
  }

  Future<void> _apply(Contact u) async {
    try {
      await Api.post('/friends/apply', {'toUid': u.uid, 'message': '你好，我想添加你为联系人'});
      if (!mounted) return;
      Navigator.pop(context);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('已向 ${u.nickname} 发送好友申请')));
    } catch (e) {
      _fail(e is ApiException ? e.message : '好友申请发送失败');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        title: const Text('扫一扫'),
        actions: [
          IconButton(
            tooltip: _torch ? '关闭手电筒' : '打开手电筒',
            onPressed: () async {
              await _controller.toggleTorch();
              if (mounted) setState(() => _torch = !_torch);
            },
            icon: Icon(_torch ? Icons.flashlight_on_rounded : Icons.flashlight_off_rounded),
          ),
        ],
      ),
      body: Stack(
        children: [
          Positioned.fill(
            child: MobileScanner(controller: _controller, onDetect: _onDetect),
          ),
          Positioned.fill(child: _mask()),
          Align(
            alignment: const Alignment(0, 0.82),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(_hint,
                    style: const TextStyle(color: Colors.white70, fontSize: 13, decoration: TextDecoration.none)),
                const SizedBox(height: 6),
                const Text('我的名片二维码可在「我的」页中打开查看',
                    style: TextStyle(color: Colors.white38, fontSize: 11.5, decoration: TextDecoration.none)),
              ],
            ),
          ),
        ],
      ),
    );
  }

  /// 取景框：四周半透明遮罩 + 白色四角
  Widget _mask() {
    const boxSize = 240.0;
    return LayoutBuilder(builder: (context, c) {
      final left = (c.maxWidth - boxSize) / 2;
      final top = (c.maxHeight - boxSize) / 2 - 40;
      return Stack(
        children: [
          Positioned(
            left: 0,
            right: 0,
            top: 0,
            height: top,
            child: const ColoredBox(color: Color(0x80000000)),
          ),
          Positioned(
            left: 0,
            right: 0,
            top: top + boxSize,
            bottom: 0,
            child: const ColoredBox(color: Color(0x80000000)),
          ),
          Positioned(
            left: 0,
            top: top,
            width: left,
            height: boxSize,
            child: const ColoredBox(color: Color(0x80000000)),
          ),
          Positioned(
            left: left + boxSize,
            right: 0,
            top: top,
            height: boxSize,
            child: const ColoredBox(color: Color(0x80000000)),
          ),
          Positioned(
            left: left,
            top: top,
            width: boxSize,
            height: boxSize,
            child: Container(
              decoration: BoxDecoration(
                border: Border.all(color: Colors.white24, width: 1),
                borderRadius: BorderRadius.circular(14),
              ),
            ),
          ),
        ],
      );
    });
  }
}

/// 扫码结果：用户资料 + 添加联系人
class _ScanResultSheet extends StatelessWidget {
  final Contact user;
  final bool isFriend;
  const _ScanResultSheet({required this.user, required this.isFriend});

  @override
  Widget build(BuildContext context) {
    final sub = [user.orgName.isNotEmpty ? user.orgName : user.deptName, user.title, '@${user.username}']
        .where((s) => s.isNotEmpty)
        .join(' · ');
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 18, 20, 20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 36,
              height: 4,
              decoration: BoxDecoration(color: const Color(0xFFE3E7EF), borderRadius: BorderRadius.circular(2)),
            ),
            const SizedBox(height: 18),
            Avatar(name: user.nickname, url: user.avatar, size: 64),
            const SizedBox(height: 10),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Flexible(
                  child: Text(user.nickname,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
                ),
                if (user.external) ...[
                  const SizedBox(width: 6),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                        color: const Color(0xFFFFF3EC), borderRadius: BorderRadius.circular(5)),
                    child: const Text('外部联系人',
                        style: TextStyle(fontSize: 10, color: Color(0xFFFF7A45), fontWeight: FontWeight.w600)),
                  ),
                ],
              ],
            ),
            if (sub.isNotEmpty) ...[
              const SizedBox(height: 5),
              Text(sub,
                  maxLines: 2,
                  textAlign: TextAlign.center,
                  style: const TextStyle(fontSize: 12.5, color: Color(0xFF8A95AA))),
            ],
            const SizedBox(height: 20),
            SizedBox(
              width: double.infinity,
              child: isFriend
                  ? OutlinedButton(
                      onPressed: () => Navigator.pop(context),
                      style: OutlinedButton.styleFrom(minimumSize: const Size.fromHeight(44)),
                      child: const Text('已是联系人'),
                    )
                  : FilledButton(
                      onPressed: () => Navigator.pop(context, 'add'),
                      style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(44)),
                      child: const Text('添加为联系人'),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
