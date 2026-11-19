import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../app/app_state.dart';
import '../core/api.dart';
import '../widgets/avatar.dart';

class EditProfilePage extends StatefulWidget {
  const EditProfilePage({super.key});

  @override
  State<EditProfilePage> createState() => _EditProfilePageState();
}

class _EditProfilePageState extends State<EditProfilePage> {
  late final TextEditingController _nickname;
  late final TextEditingController _signature;
  int _gender = 0;
  String _avatar = '';
  bool _uploading = false;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    final u = context.read<AppState>().user ?? {};
    _nickname = TextEditingController(text: '${u['nickname'] ?? ''}');
    _signature = TextEditingController(text: '${u['signature'] ?? ''}');
    _gender = (u['gender'] as num?)?.toInt() ?? 0;
    _avatar = '${u['avatar'] ?? ''}';
  }

  @override
  void dispose() {
    _nickname.dispose();
    _signature.dispose();
    super.dispose();
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _pickAvatar() async {
    if (_uploading) return;
    final picked = await ImagePicker().pickImage(source: ImageSource.gallery, maxWidth: 800, imageQuality: 85);
    if (picked == null) return;
    setState(() => _uploading = true);
    try {
      final data = await Api.upload('/files/upload', picked.path);
      final fileId = '${data['fileId'] ?? ''}';
      if (fileId.isEmpty) throw ApiException('上传失败');
      setState(() => _avatar = 'file:$fileId');
      _toast('头像已上传，记得保存');
    } catch (e) {
      _toast('$e');
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _save() async {
    final nickname = _nickname.text.trim();
    if (nickname.isEmpty) {
      _toast('昵称不能为空');
      return;
    }
    setState(() => _saving = true);
    try {
      await context.read<AppState>().updateProfile({
        'nickname': nickname,
        'signature': _signature.text.trim(),
        'gender': _gender,
        'avatar': _avatar,
      });
      if (!mounted) return;
      Navigator.pop(context);
    } catch (e) {
      _toast('$e');
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    return Scaffold(
      appBar: AppBar(
        title: const Text('编辑资料'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _save,
            child: Text(_saving ? '保存中…' : '保存'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Row(
            children: [
              Avatar(name: _nickname.text.isEmpty ? '我' : _nickname.text, url: _avatar, size: 64),
              const SizedBox(width: 14),
              OutlinedButton(
                onPressed: _uploading ? null : _pickAvatar,
                child: Text(_uploading ? '上传中…' : '更换头像'),
              ),
            ],
          ),
          const SizedBox(height: 22),
          TextField(
            controller: _nickname,
            maxLength: 32,
            decoration: const InputDecoration(labelText: '昵称', hintText: '请输入昵称'),
            onChanged: (_) => setState(() {}),
          ),
          const SizedBox(height: 6),
          Row(
            children: [
              const Text('性别', style: TextStyle(color: Colors.grey)),
              const SizedBox(width: 16),
              for (final g in const [(0, '保密'), (1, '男'), (2, '女')])
                Padding(
                  padding: const EdgeInsets.only(right: 8),
                  child: ChoiceChip(
                    label: Text(g.$2),
                    selected: _gender == g.$1,
                    onSelected: (_) => setState(() => _gender = g.$1),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _signature,
            maxLength: 120,
            maxLines: 3,
            decoration: const InputDecoration(labelText: '个性签名', hintText: '介绍一下自己吧'),
          ),
          const SizedBox(height: 12),
          Text(
            '@${app.user?['username'] ?? ''} · UID ${app.user?['uid'] ?? ''}',
            style: const TextStyle(color: Colors.grey, fontSize: 12),
          ),
        ],
      ),
    );
  }
}
