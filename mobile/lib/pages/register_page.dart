import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import 'legal_page.dart';

class RegisterPage extends StatefulWidget {
  /// 登录页当前填写的服务地址，注册时沿用
  final String server;
  const RegisterPage({super.key, this.server = ''});

  @override
  State<RegisterPage> createState() => _RegisterPageState();
}

class _RegisterPageState extends State<RegisterPage> {
  final _username = TextEditingController();
  final _nickname = TextEditingController();
  final _password = TextEditingController();
  final _confirm = TextEditingController();
  late final TextEditingController _server;
  bool _loading = false;
  bool _showServer = false;
  String _error = '';

  @override
  void initState() {
    super.initState();
    final app = context.read<AppState>();
    _server = TextEditingController(text: widget.server.isNotEmpty ? widget.server : app.apiBase);
  }

  @override
  void dispose() {
    _username.dispose();
    _nickname.dispose();
    _password.dispose();
    _confirm.dispose();
    _server.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_loading) return;
    final username = _username.text.trim();
    final password = _password.text;
    if (username.length < 3) {
      setState(() => _error = '用户名至少 3 位');
      return;
    }
    if (password.length < 6) {
      setState(() => _error = '密码至少 6 位');
      return;
    }
    if (password != _confirm.text) {
      setState(() => _error = '两次输入的密码不一致');
      return;
    }
    final app = context.read<AppState>();
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      await app.setServer(_server.text);
      await app.register(username, password, _nickname.text.trim());
      // 注册成功后 AppState 已切换为登录态，关闭注册页回到首页
      if (mounted) Navigator.of(context).pop();
    } catch (e) {
      if (mounted) setState(() => _error = e is ApiException ? e.message : '$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Scaffold(
      appBar: AppBar(
        title: const Text('注册账号'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, size: 18),
          onPressed: () => Navigator.of(context).maybePop(),
        ),
      ),
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Center(
                child: Container(
                  width: 68,
                  height: 68,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [Color(0xFF4F6EF7), Color(0xFF2F8FF5)],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: const Text('N',
                      style: TextStyle(color: Colors.white, fontSize: 34, fontWeight: FontWeight.w700)),
                ),
              ),
              const SizedBox(height: 14),
              const Center(
                child: Text('创建 NexIM 账号', style: TextStyle(fontSize: 20, fontWeight: FontWeight.w700)),
              ),
              const SizedBox(height: 26),
              TextField(
                controller: _username,
                textInputAction: TextInputAction.next,
                autocorrect: false,
                decoration: const InputDecoration(
                  labelText: '用户名',
                  hintText: '至少 3 位，注册后用于登录',
                  prefixIcon: Icon(Icons.person_outline),
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _nickname,
                textInputAction: TextInputAction.next,
                decoration: const InputDecoration(
                  labelText: '昵称（选填）',
                  hintText: '留空则与用户名相同',
                  prefixIcon: Icon(Icons.badge_outlined),
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _password,
                obscureText: true,
                textInputAction: TextInputAction.next,
                decoration: const InputDecoration(
                  labelText: '密码',
                  hintText: '至少 6 位',
                  prefixIcon: Icon(Icons.lock_outline),
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _confirm,
                obscureText: true,
                onSubmitted: (_) => _submit(),
                decoration: const InputDecoration(
                  labelText: '确认密码',
                  prefixIcon: Icon(Icons.lock_reset_outlined),
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 6),
              Align(
                alignment: Alignment.centerRight,
                child: TextButton(
                  onPressed: () => setState(() => _showServer = !_showServer),
                  child: const Text('服务器设置'),
                ),
              ),
              if (_showServer)
                TextField(
                  controller: _server,
                  keyboardType: TextInputType.url,
                  decoration: const InputDecoration(
                    labelText: '服务地址',
                    hintText: 'http://192.168.1.10:8080/api/v1',
                    prefixIcon: Icon(Icons.dns_outlined),
                    border: OutlineInputBorder(),
                  ),
                ),
              if (_error.isNotEmpty) ...[
                const SizedBox(height: 12),
                Text(_error, style: TextStyle(color: scheme.error)),
              ],
              const SizedBox(height: 18),
              SizedBox(
                height: 48,
                child: FilledButton(
                  onPressed: _loading ? null : _submit,
                  child: _loading
                      ? const SizedBox(
                          width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                      : const Text('注册并登录', style: TextStyle(fontSize: 16)),
                ),
              ),
              const SizedBox(height: 10),
              _consent(),
              Center(
                child: TextButton(
                  onPressed: () => Navigator.of(context).maybePop(),
                  child: const Text('已有账号？返回登录'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _consent() {
    const normal = TextStyle(fontSize: 12, color: Color(0xFF9AA3B2));
    const link = TextStyle(fontSize: 12, color: Color(0xFF3D6CF6));
    return Wrap(
      alignment: WrapAlignment.center,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        const Text('注册即代表同意', style: normal),
        GestureDetector(
          onTap: () => Navigator.of(context)
              .push(MaterialPageRoute(builder: (_) => const LegalPage(service: true))),
          child: const Text('《服务协议》', style: link),
        ),
        const Text('与', style: normal),
        GestureDetector(
          onTap: () => Navigator.of(context)
              .push(MaterialPageRoute(builder: (_) => const LegalPage(service: false))),
          child: const Text('《隐私政策》', style: link),
        ),
      ],
    );
  }
}
