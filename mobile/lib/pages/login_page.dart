import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import 'legal_page.dart';
import 'register_page.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  late final TextEditingController _account;
  late final TextEditingController _password;
  final _code = TextEditingController();
  late final TextEditingController _server;
  bool _loading = false;
  String _error = '';
  bool _showServer = false;
  // 登录方式：false 密码登录，true 验证码登录
  bool _codeMode = false;
  bool _sendingCode = false;
  int _countdown = 0;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    final app = context.read<AppState>();
    _account = TextEditingController(text: 'linxiao');
    _password = TextEditingController(text: '123456');
    _server = TextEditingController(text: app.apiBase);
  }

  @override
  void dispose() {
    _timer?.cancel();
    _account.dispose();
    _password.dispose();
    _code.dispose();
    _server.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_loading) return;
    final app = context.read<AppState>();
    final account = _account.text.trim();
    if (account.isEmpty) {
      setState(() => _error = '请输入账号');
      return;
    }
    if (_codeMode && _code.text.trim().isEmpty) {
      setState(() => _error = '请输入验证码');
      return;
    }
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      await app.setServer(_server.text);
      if (_codeMode) {
        await app.loginWithCode(account, _code.text.trim());
      } else {
        await app.login(account, _password.text);
      }
    } catch (e) {
      if (mounted) setState(() => _error = e is ApiException ? e.message : '$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  /// 发送验证码并开始 60 秒倒计时
  Future<void> _sendCode() async {
    if (_sendingCode || _countdown > 0) return;
    final account = _account.text.trim();
    if (account.isEmpty) {
      setState(() => _error = '请先输入手机号或账号');
      return;
    }
    setState(() {
      _sendingCode = true;
      _error = '';
    });
    try {
      await context.read<AppState>().setServer(_server.text);
      final data = await Api.post('/auth/sms-code', {'target': account, 'scene': 'login'});
      final demo = '${data['demoCode'] ?? ''}';
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(demo.isEmpty ? '验证码已发送' : '验证码已发送，演示验证码：$demo'),
      ));
      setState(() => _countdown = 60);
      _timer?.cancel();
      _timer = Timer.periodic(const Duration(seconds: 1), (t) {
        if (!mounted) {
          t.cancel();
          return;
        }
        setState(() {
          _countdown--;
          if (_countdown <= 0) t.cancel();
        });
      });
    } catch (e) {
      if (mounted) setState(() => _error = e is ApiException ? e.message : '$e');
    } finally {
      if (mounted) setState(() => _sendingCode = false);
    }
  }

  void _switchMode(bool codeMode) {
    if (_loading || _codeMode == codeMode) return;
    setState(() {
      _codeMode = codeMode;
      _error = '';
    });
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 28, vertical: 32),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Center(
                  child: Container(
                    width: 76,
                    height: 76,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      gradient: const LinearGradient(
                        colors: [Color(0xFF4F6EF7), Color(0xFF2F8FF5)],
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                      ),
                      borderRadius: BorderRadius.circular(22),
                    ),
                    child: const Text('N',
                        style: TextStyle(color: Colors.white, fontSize: 40, fontWeight: FontWeight.w700)),
                  ),
                ),
                const SizedBox(height: 18),
                const Center(
                  child: Text('NexIM', style: TextStyle(fontSize: 26, fontWeight: FontWeight.w700)),
                ),
                const Center(
                  child: Text('企业即时通讯', style: TextStyle(fontSize: 13, color: Colors.grey)),
                ),
                const SizedBox(height: 26),
                _modeTabs(),
                const SizedBox(height: 18),
                TextField(
                  controller: _account,
                  keyboardType: _codeMode ? TextInputType.phone : TextInputType.text,
                  textInputAction: TextInputAction.next,
                  decoration: InputDecoration(
                    labelText: _codeMode ? '手机号 / 账号' : '账号',
                    hintText: _codeMode ? '请输入手机号或账号' : null,
                    prefixIcon: const Icon(Icons.person_outline),
                    border: const OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 14),
                if (_codeMode)
                  Row(
                    children: [
                      Expanded(
                        child: TextField(
                          controller: _code,
                          keyboardType: TextInputType.number,
                          onSubmitted: (_) => _submit(),
                          decoration: const InputDecoration(
                            labelText: '验证码',
                            prefixIcon: Icon(Icons.verified_user_outlined),
                            border: OutlineInputBorder(),
                          ),
                        ),
                      ),
                      const SizedBox(width: 10),
                      SizedBox(
                        height: 56,
                        child: OutlinedButton(
                          onPressed: (_sendingCode || _countdown > 0 || _loading) ? null : _sendCode,
                          child: Text(
                            _countdown > 0 ? '${_countdown}s 后重发' : '获取验证码',
                            style: const TextStyle(fontSize: 13),
                          ),
                        ),
                      ),
                    ],
                  )
                else
                  TextField(
                    controller: _password,
                    obscureText: true,
                    onSubmitted: (_) => _submit(),
                    decoration: const InputDecoration(
                      labelText: '密码',
                      prefixIcon: Icon(Icons.lock_outline),
                      border: OutlineInputBorder(),
                    ),
                  ),
                const SizedBox(height: 8),
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
                        : const Text('登录', style: TextStyle(fontSize: 16)),
                  ),
                ),
                const SizedBox(height: 12),
                _consent(),
                const SizedBox(height: 4),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Text('还没有账号？', style: TextStyle(fontSize: 13, color: Colors.grey)),
                    TextButton(
                      onPressed: _loading
                          ? null
                          : () => Navigator.of(context).push(
                                MaterialPageRoute(builder: (_) => RegisterPage(server: _server.text)),
                              ),
                      child: const Text('立即注册'),
                    ),
                  ],
                ),
                if (!_codeMode) ...[
                  const SizedBox(height: 4),
                  const Center(
                    child: Text('默认演示账号 linxiao / 123456',
                        style: TextStyle(fontSize: 12, color: Colors.grey)),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _modeTabs() {
    return Container(
      height: 42,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: const Color(0xFFF1F3F8),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        children: [
          _tab('密码登录', !_codeMode),
          _tab('验证码登录', _codeMode),
        ],
      ),
    );
  }

  Widget _tab(String label, bool active) {
    return Expanded(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: () => _switchMode(label == '验证码登录'),
        child: Container(
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: active ? Colors.white : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
            boxShadow: active
                ? const [BoxShadow(color: Color(0x141A2850), blurRadius: 8, offset: Offset(0, 2))]
                : null,
          ),
          child: Text(
            label,
            style: TextStyle(
              fontSize: 14,
              fontWeight: active ? FontWeight.w600 : FontWeight.w400,
              color: active ? const Color(0xFF1B2333) : const Color(0xFF8A95AA),
            ),
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
        const Text('登录即代表同意', style: normal),
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
