import 'package:flutter/material.dart';

import '../core/api.dart';
import 'friend_picker_page.dart';

/// 发起群聊：填写群名称并选择成员
class CreateGroupPage extends StatefulWidget {
  const CreateGroupPage({super.key});

  @override
  State<CreateGroupPage> createState() => _CreateGroupPageState();
}

class _CreateGroupPageState extends State<CreateGroupPage> {
  final _name = TextEditingController();
  List<String> _members = [];
  bool _submitting = false;

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  Future<void> _pick() async {
    final picked = await Navigator.of(context).push<List<String>>(
      MaterialPageRoute(builder: (_) => const FriendPickerPage(title: '选择群成员')),
    );
    if (picked == null || !mounted) return;
    setState(() => _members = picked);
  }

  Future<void> _create() async {
    final name = _name.text.trim();
    if (name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('请输入群名称')));
      return;
    }
    setState(() => _submitting = true);
    try {
      final g = await Api.post('/groups', {'name': name, 'members': _members});
      if (!mounted) return;
      Navigator.of(context).pop(g is Map ? Map<String, dynamic>.from(g) : <String, dynamic>{});
    } catch (e) {
      if (!mounted) return;
      setState(() => _submitting = false);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('创建失败：$e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFFF5F6FA),
      appBar: AppBar(
        backgroundColor: Colors.white,
        elevation: 0,
        centerTitle: true,
        title: const Text('发起群聊',
            style: TextStyle(fontSize: 17, fontWeight: FontWeight.w700, color: Color(0xFF1B2333))),
        iconTheme: const IconThemeData(color: Color(0xFF1B2333)),
      ),
      body: Column(
        children: [
          Container(
            color: Colors.white,
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
            child: Row(
              children: [
                const Text('群名称', style: TextStyle(fontSize: 15, color: Color(0xFF1B2333))),
                const SizedBox(width: 14),
                Expanded(
                  child: TextField(
                    controller: _name,
                    maxLength: 32,
                    decoration: const InputDecoration(
                      isDense: true,
                      counterText: '',
                      border: InputBorder.none,
                      hintText: '请输入群名称',
                      hintStyle: TextStyle(fontSize: 15, color: Color(0xFFADB3C1)),
                    ),
                    style: const TextStyle(fontSize: 15),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 10),
          Container(
            color: Colors.white,
            child: ListTile(
              leading: const Icon(Icons.group_add_outlined, color: Color(0xFF1B2333)),
              title: const Text('群成员', style: TextStyle(fontSize: 15)),
              subtitle: Text(_members.isEmpty ? '未选择' : '已选 ${_members.length} 人',
                  style: const TextStyle(fontSize: 12.5, color: Color(0xFF9AA5B8))),
              trailing: const Icon(Icons.chevron_right, color: Color(0xFFADB3C1)),
              onTap: _pick,
            ),
          ),
          const Spacer(),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
              child: SizedBox(
                width: double.infinity,
                height: 46,
                child: FilledButton(
                  onPressed: _submitting ? null : _create,
                  style: FilledButton.styleFrom(
                    backgroundColor: const Color(0xFF2F7CF6),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(23)),
                  ),
                  child: Text(_submitting ? '创建中…' : '创建群聊',
                      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
