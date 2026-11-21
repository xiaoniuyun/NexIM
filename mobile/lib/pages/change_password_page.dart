import 'package:flutter/material.dart';

import '../core/api.dart';

/// 修改密码：校验原密码，成功后提示重新登录（旧令牌仍有效，退出后需用新密码登录）
class ChangePasswordPage extends StatefulWidget {
  const ChangePasswordPage({super.key});

  @override
  State<ChangePasswordPage> createState() => _ChangePasswordPageState();
}

class _ChangePasswordPageState extends State<ChangePasswordPage> {
  final _old = TextEditingController();
  final _pwd = TextEditingController();
  final _confirm = TextEditingController();
  bool _saving = false;
  bool _obscure = true;

  @override
  void dispose() {
    _old.dispose();
    _pwd.dispose();
    _confirm.dispose();
    super.dispose();
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _submit() async {
    final oldPwd = _old.text;
    final pwd = _pwd.text;
    if (oldPwd.isEmpty) {
      _toast('请输入原密码');
      return;
    }
    if (pwd.length < 6) {
      _toast('新密码至少 6 位');
      return;
    }
    if (pwd == oldPwd) {
      _toast('新密码不能与原密码相同');
      return;
    }
    if (pwd != _confirm.text) {
      _toast('两次输入的新密码不一致');
      return;
    }
    setState(() => _saving = true);
    try {
      await Api.post('/auth/password', {'oldPassword': oldPwd, 'newPassword': pwd});
      if (!mounted) return;
      _toast('密码修改成功，请使用新密码登录');
      Navigator.pop(context);
    } catch (e) {
      _toast('$e');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('修改密码')),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 18, 20, 24),
        children: [
          TextField(
            controller: _old,
            obscureText: _obscure,
            enableSuggestions: false,
            autocorrect: false,
            decoration: const InputDecoration(labelText: '原密码', hintText: '请输入当前登录密码'),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: _pwd,
            obscureText: _obscure,
            enableSuggestions: false,
            autocorrect: false,
            decoration: const InputDecoration(labelText: '新密码', hintText: '至少 6 位'),
          ),
          const SizedBox(height: 10),
          TextField(
            controller: _confirm,
            obscureText: _obscure,
            enableSuggestions: false,
            autocorrect: false,
            decoration: const InputDecoration(labelText: '确认新密码', hintText: '再次输入新密码'),
          ),
          const SizedBox(height: 6),
          Row(
            children: [
              Checkbox(
                value: !_obscure,
                onChanged: (v) => setState(() => _obscure = !(v ?? false)),
              ),
              const Text('显示密码', style: TextStyle(fontSize: 13, color: Color(0xFF5B6880))),
            ],
          ),
          const SizedBox(height: 14),
          FilledButton(
            style: FilledButton.styleFrom(minimumSize: const Size.fromHeight(48)),
            onPressed: _saving ? null : _submit,
            child: Text(_saving ? '提交中…' : '确认修改'),
          ),
          const SizedBox(height: 14),
          const Text('修改成功后，当前登录状态仍然有效；如需在其他设备生效，请重新登录。',
              style: TextStyle(fontSize: 12, color: Color(0xFF8A95AA))),
        ],
      ),
    );
  }
}
