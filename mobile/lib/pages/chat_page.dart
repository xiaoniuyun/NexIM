import 'dart:async';
import 'dart:io';

import 'package:audioplayers/audioplayers.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';
import 'package:record/record.dart';

import '../app/app_state.dart';
import '../app/call_state.dart';
import '../core/api.dart';
import '../core/models.dart';
import '../core/util.dart';
import '../widgets/avatar.dart';

class ChatPage extends StatefulWidget {
  final Conv conv;
  const ChatPage({super.key, required this.conv});

  @override
  State<ChatPage> createState() => _ChatPageState();
}

class _ChatPageState extends State<ChatPage> {
  final _input = TextEditingController();
  final _scroll = ScrollController();
  bool _uploading = false;
  int _lastCount = -1;
  late final AppState _app;
  // 语音录制与播放
  final AudioRecorder _recorder = AudioRecorder();
  final AudioPlayer _player = AudioPlayer();
  bool _recording = false;
  int _recSec = 0;
  DateTime? _recStart;
  Timer? _recTimer;
  String? _playingId;
  StreamSubscription? _playerSub;

  @override
  void initState() {
    super.initState();
    _app = context.read<AppState>();
    _app.setChatVisible(true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _app.openConv(widget.conv.convId);
    });
  }

  @override
  void dispose() {
    _playerSub?.cancel();
    _recTimer?.cancel();
    _recorder.dispose();
    _player.dispose();
    _app.setChatVisible(false);
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }

  // ===== 语音消息 =====
  Future<void> _toggleRecord() async {
    if (_recording) {
      await _stopRecord();
      return;
    }
    if (!await _recorder.hasPermission()) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('没有麦克风权限')));
      }
      return;
    }
    try {
      final dir = await getTemporaryDirectory();
      final path = '${dir.path}/voice_${DateTime.now().millisecondsSinceEpoch}.m4a';
      await _recorder.start(
        const RecordConfig(encoder: AudioEncoder.aacLc, bitRate: 96000, sampleRate: 44100),
        path: path,
      );
      _recStart = DateTime.now();
      if (mounted) setState(() { _recording = true; _recSec = 0; });
      _recTimer = Timer.periodic(const Duration(seconds: 1), (_) {
        if (mounted) setState(() => _recSec++);
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('录音失败：$e')));
      }
    }
  }

  Future<void> _stopRecord() async {
    _recTimer?.cancel();
    _recTimer = null;
    final path = await _recorder.stop();
    if (mounted) setState(() => _recording = false);
    if (path == null) return;
    final secs = ((DateTime.now().difference(_recStart ?? DateTime.now()).inMilliseconds) / 1000).round().clamp(1, 600);
    final file = File(path);
    final size = await file.length();
    if (size < 800) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('录音时间太短')));
      }
      try { await file.delete(); } catch (_) {}
      return;
    }
    setState(() => _uploading = true);
    try {
      final rec = await Api.upload('/files/upload', path, fields: {'convId': widget.conv.convId});
      _app.sendMedia('voice', '${rec['name'] ?? '语音消息'}', {
        'fileId': rec['fileId'],
        'name': rec['name'] ?? '语音消息',
        'size': rec['size'] ?? size,
        'mime': rec['mimeType'] ?? 'audio/mp4',
        'duration': secs,
      });
      _scrollToBottom();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('语音发送失败：$e')));
      }
    } finally {
      try { await file.delete(); } catch (_) {}
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _togglePlay(Msg m) async {
    final extra = m.extra is Map ? Map<String, dynamic>.from(m.extra) : <String, dynamic>{};
    final fid = '${extra['fileId'] ?? ''}';
    if (fid.isEmpty) return;
    if (_playingId == m.msgId) {
      await _player.stop();
      if (mounted) setState(() => _playingId = null);
      return;
    }
    try {
      await _player.stop();
      await _player.play(UrlSource(Api.fileUrl(fid)));
      _playerSub?.cancel();
      _playerSub = _player.onPlayerComplete.listen((_) {
        if (mounted) setState(() => _playingId = null);
      });
      if (mounted) setState(() => _playingId = m.msgId);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('播放失败：$e')));
      }
    }
  }

  Future<void> _send() async {
    final text = _input.text.trim();
    if (text.isEmpty) return;
    context.read<AppState>().sendText(text);
    _input.clear();
    _scrollToBottom();
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.jumpTo(_scroll.position.maxScrollExtent);
      }
    });
  }

  Future<void> _upload(String path, String name) async {
    final app = context.read<AppState>();
    setState(() => _uploading = true);
    try {
      final rec = await Api.upload('/files/upload', path, fields: {'convId': widget.conv.convId});
      final mime = '${rec['mimeType'] ?? ''}';
      final type = mime.startsWith('image/')
          ? 'image'
          : mime.startsWith('video/')
              ? 'video'
              : 'file';
      app.sendMedia(type, '${rec['name'] ?? name}', {
        'fileId': rec['fileId'],
        'name': rec['name'] ?? name,
        'size': rec['size'] ?? 0,
        'mime': mime,
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('上传失败：$e')));
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  void _pickAttachment() {
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Row(
          mainAxisAlignment: MainAxisAlignment.spaceAround,
          children: [
            _attachAction(ctx, Icons.photo_camera_outlined, '拍照', () async {
              final x = await ImagePicker().pickImage(source: ImageSource.camera, imageQuality: 85);
              if (x != null) _upload(x.path, x.name);
            }),
            _attachAction(ctx, Icons.photo_library_outlined, '相册', () async {
              final x = await ImagePicker().pickImage(source: ImageSource.gallery, imageQuality: 85);
              if (x != null) _upload(x.path, x.name);
            }),
            _attachAction(ctx, Icons.videocam_outlined, '视频', () async {
              final x = await ImagePicker().pickVideo(source: ImageSource.gallery);
              if (x != null) _upload(x.path, x.name);
            }),
            _attachAction(ctx, Icons.attach_file, '文件', () async {
              final f = await FilePicker.pickFile();
              if (f != null && f.path != null) _upload(f.path!, f.name);
            }),
          ],
        ),
      ),
    );
  }

  Widget _attachAction(BuildContext sheetCtx, IconData icon, String label, Future<void> Function() run) {
    return InkWell(
      onTap: () {
        Navigator.pop(sheetCtx);
        run();
      },
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 28),
            const SizedBox(height: 6),
            Text(label, style: const TextStyle(fontSize: 12)),
          ],
        ),
      ),
    );
  }

  void _msgActions(Msg m) {
    final app = context.read<AppState>();
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (m.type == 'text')
              ListTile(
                leading: const Icon(Icons.copy),
                title: const Text('复制'),
                onTap: () {
                  Clipboard.setData(ClipboardData(text: m.content));
                  Navigator.pop(ctx);
                },
              ),
            ListTile(
              leading: const Icon(Icons.star_border),
              title: const Text('收藏'),
              onTap: () async {
                Navigator.pop(ctx);
                try {
                  await Api.post('/favorites', {'msgId': m.msgId});
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('已收藏')));
                  }
                } catch (e) {
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
                  }
                }
              },
            ),
            if (m.mine && !m.pending)
              ListTile(
                leading: const Icon(Icons.undo),
                title: const Text('撤回'),
                onTap: () async {
                  Navigator.pop(ctx);
                  try {
                    await Api.post('/messages/revoke', {'msgId': m.msgId});
                    app.markRevoked(m.msgId);
                  } catch (e) {
                    if (mounted) {
                      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
                    }
                  }
                },
              ),
            ListTile(
              leading: const Icon(Icons.delete_outline, color: Colors.red),
              title: const Text('删除（仅本机）', style: TextStyle(color: Colors.red)),
              onTap: () {
                Navigator.pop(ctx);
                app.removeLocal(m.msgId);
              },
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppState>();
    final call = context.watch<CallState>();
    final list = app.messages.where((m) => m.convId == widget.conv.convId).toList();
    if (list.length != _lastCount) {
      _lastCount = list.length;
      _scrollToBottom();
    }

    return Scaffold(
      appBar: AppBar(
        titleSpacing: 0,
        title: Row(
          children: [
            Avatar(name: widget.conv.title, url: widget.conv.peerAvatar, size: 34),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(widget.conv.title,
                      maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 16)),
                  Text(
                    widget.conv.isGroup ? '群聊' : (app.connected ? '在线' : '连接中'),
                    style: const TextStyle(fontSize: 11, color: Colors.grey, fontWeight: FontWeight.normal),
                  ),
                ],
              ),
            ),
          ],
        ),
        actions: [
          if (!widget.conv.isGroup) ...[
            IconButton(
              tooltip: '语音通话',
              onPressed: call.busy ? null : () => call.startCall(widget.conv, 'audio'),
              icon: const Icon(Icons.call_outlined),
            ),
            IconButton(
              tooltip: '视频通话',
              onPressed: call.busy ? null : () => call.startCall(widget.conv, 'video'),
              icon: const Icon(Icons.videocam_outlined),
            ),
          ],
          PopupMenuButton<String>(
            onSelected: (v) {
              if (v == 'pin') app.togglePinned(widget.conv);
              if (v == 'mute') app.toggleMuted(widget.conv);
            },
            itemBuilder: (_) => [
              PopupMenuItem(value: 'pin', child: Text(widget.conv.pinned ? '取消置顶' : '置顶会话')),
              PopupMenuItem(value: 'mute', child: Text(widget.conv.muted ? '开启提醒' : '消息免打扰')),
            ],
          ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: list.isEmpty
                ? const Center(child: Text('暂无消息', style: TextStyle(color: Colors.grey)))
                : ListView.builder(
                    controller: _scroll,
                    padding: const EdgeInsets.fromLTRB(12, 12, 12, 12),
                    itemCount: list.length,
                    itemBuilder: (context, i) => _messageTile(app, list[i]),
                  ),
          ),
          if (_uploading) const LinearProgressIndicator(minHeight: 2),
          _inputBar(),
        ],
      ),
    );
  }

  Widget _messageTile(AppState app, Msg m) {
    if (m.type == 'system') {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Center(
          child: Text(m.content.isEmpty ? '系统消息' : m.content,
              style: const TextStyle(fontSize: 12, color: Colors.grey)),
        ),
      );
    }
    final mine = m.mine;
    final extra = m.extra is Map ? Map<String, dynamic>.from(m.extra) : <String, dynamic>{};
    final bubbleColor = mine ? const Color(0xFF4F6EF7) : const Color(0xFFF1F3F6);
    final textColor = mine ? Colors.white : const Color(0xFF1F2329);

    Widget content;
    if (m.status == 2) {
      content = Text('[消息已撤回]', style: TextStyle(fontSize: 12, color: mine ? Colors.white70 : Colors.grey));
    } else if (m.type == 'image') {
      final fid = '${extra['fileId'] ?? ''}';
      content = fid.isEmpty
          ? Text('[图片]', style: TextStyle(color: textColor))
          : ClipRRect(
              borderRadius: BorderRadius.circular(8),
              child: Image.network(
                Api.fileUrl(fid),
                width: 190,
                fit: BoxFit.cover,
                errorBuilder: (_, __, ___) => const Padding(
                  padding: EdgeInsets.all(8),
                  child: Text('[图片加载失败]', style: TextStyle(color: Colors.grey, fontSize: 12)),
                ),
              ),
            );
    } else if (m.type == 'video') {
      content = Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.play_circle_fill, color: mine ? Colors.white : const Color(0xFF4F6EF7), size: 30),
          const SizedBox(width: 8),
          Text('视频 ${extra['name'] ?? ''}', style: TextStyle(color: textColor, fontSize: 13)),
        ],
      );
    } else if (m.type == 'file') {
      content = Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.insert_drive_file_outlined, color: mine ? Colors.white : const Color(0xFF4F6EF7)),
          const SizedBox(width: 8),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 190),
                child: Text('${extra['name'] ?? '文件'}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: textColor, fontSize: 13)),
              ),
              Text(
                bytes((extra['size'] as num?)?.toInt() ?? 0),
                style: TextStyle(color: mine ? Colors.white70 : Colors.grey, fontSize: 11),
              ),
            ],
          ),
        ],
      );
    } else if (m.type == 'voice') {
      final dur = (extra['duration'] as num?)?.toInt() ?? 0;
      final fid = '${extra['fileId'] ?? ''}';
      final playing = _playingId == m.msgId;
      content = InkWell(
        onTap: fid.isEmpty ? null : () => _togglePlay(m),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(playing ? Icons.pause_circle_filled : Icons.play_circle_fill, color: textColor, size: 28),
            const SizedBox(width: 6),
            Text(dur > 0 ? '$dur″' : '语音', style: TextStyle(color: textColor, fontSize: 14)),
            const SizedBox(width: 8),
            Icon(Icons.graphic_eq, color: textColor, size: 18),
          ],
        ),
      );
    } else {
      content = Text(m.content, style: TextStyle(color: textColor, fontSize: 15, height: 1.35));
    }

    final bubble = Container(
      constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.72),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
      decoration: BoxDecoration(
        color: bubbleColor,
        borderRadius: BorderRadius.only(
          topLeft: const Radius.circular(10),
          topRight: const Radius.circular(10),
          bottomLeft: Radius.circular(mine ? 10 : 2),
          bottomRight: Radius.circular(mine ? 2 : 10),
        ),
      ),
      child: content,
    );

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        mainAxisAlignment: mine ? MainAxisAlignment.end : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!mine) ...[
            Avatar(
              name: m.fromName.isNotEmpty ? m.fromName : widget.conv.title,
              url: widget.conv.isGroup ? '' : widget.conv.peerAvatar,
              size: 34,
            ),
            const SizedBox(width: 8),
          ],
          Flexible(
            child: Column(
              crossAxisAlignment: mine ? CrossAxisAlignment.end : CrossAxisAlignment.start,
              children: [
                if (!mine && widget.conv.isGroup)
                  Padding(
                    padding: const EdgeInsets.only(left: 2, bottom: 3),
                    child: Text(m.fromName,
                        style: const TextStyle(fontSize: 11, color: Colors.grey)),
                  ),
                GestureDetector(onLongPress: () => _msgActions(m), child: bubble),
                Padding(
                  padding: const EdgeInsets.only(top: 3, left: 2, right: 2),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      if (m.pending)
                        const Icon(Icons.schedule, size: 11, color: Colors.grey)
                      else if (m.status == 3)
                        const Icon(Icons.error_outline, size: 12, color: Colors.red),
                      if (m.pending || m.status == 3) const SizedBox(width: 3),
                      Text(timeText(m.createdAt),
                          style: const TextStyle(fontSize: 10, color: Colors.grey)),
                    ],
                  ),
                ),
              ],
            ),
          ),
          if (mine) const SizedBox(width: 4),
        ],
      ),
    );
  }

  Widget _inputBar() {
    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(6, 6, 6, 6),
        decoration: BoxDecoration(
          color: Theme.of(context).scaffoldBackgroundColor,
          border: Border(top: BorderSide(color: Colors.black.withValues(alpha: 0.06))),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            IconButton(
              onPressed: _uploading ? null : _pickAttachment,
              icon: const Icon(Icons.add_circle_outline),
            ),
            Expanded(
              child: TextField(
                controller: _input,
                minLines: 1,
                maxLines: 4,
                textInputAction: TextInputAction.newline,
                decoration: InputDecoration(
                  hintText: _recording ? '正在录音 $_recSec″，点击红色按钮结束' : '输入消息…',
                  isDense: true,
                  filled: true,
                  fillColor: const Color(0xFFF1F3F6),
                  contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(18),
                    borderSide: BorderSide.none,
                  ),
                ),
              ),
            ),
            const SizedBox(width: 4),
            IconButton(
              tooltip: _recording ? '结束录音' : '语音消息',
              onPressed: _uploading ? null : () => _toggleRecord(),
              icon: Icon(_recording ? Icons.stop_circle : Icons.mic_none),
              color: _recording ? Colors.red : null,
            ),
            IconButton.filled(
              onPressed: _send,
              icon: const Icon(Icons.send, size: 20),
              style: IconButton.styleFrom(
                backgroundColor: const Color(0xFF4F6EF7),
                foregroundColor: Colors.white,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
