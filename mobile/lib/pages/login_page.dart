import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';

class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  late final TextEditingController _account;
  late final TextEditingController _password;
  late final TextEditingController _server;
  bool _loading = false;
  String _error = '';
  bool _showServer = false;

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
    _account.dispose();
    _password.dispose();
    _server.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_loading) return;
    final app = context.read<AppState>();
    setState(() {
      _loading = true;
      _error = '';
    });
    try {
      await app.setServer(_server.text);
      await app.login(_account.text.trim(), _password.text);
    } catch (e) {
      setState(() => _error = e is ApiException ? e.message : '$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
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
                const SizedBox(height: 34),
                TextField(
                  controller: _account,
                  textInputAction: TextInputAction.next,
                  decoration: const InputDecoration(
                    labelText: '账号',
                    prefixIcon: Icon(Icons.person_outline),
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 14),
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
                const SizedBox(height: 16),
                const Center(
                  child: Text('默认演示账号 linxiao / 123456',
                      style: TextStyle(fontSize: 12, color: Colors.grey)),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
