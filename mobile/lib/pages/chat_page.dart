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
  // 工具栏面板
  bool _panelOpen = false;
  int _panelPage = 0;
  bool _emojiOpen = false;

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

  void _snack(String msg) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  void _panelAction(String key) {
    final call = context.read<CallState>();
    switch (key) {
      case 'call-audio':
      case 'call-video':
        if (widget.conv.isGroup) {
          _snack('群聊暂不支持音视频通话');
          return;
        }
        if (call.busy) {
          _snack('当前正在通话中');
          return;
        }
        setState(() => _panelOpen = false);
        call.startCall(widget.conv, key == 'call-audio' ? 'audio' : 'video');
        return;
      case 'file':
        setState(() => _panelOpen = false);
        FilePicker.pickFile().then((f) {
          if (f != null && f.path != null) _upload(f.path!, f.name);
        });
        return;
      case 'image':
        setState(() => _panelOpen = false);
        ImagePicker()
            .pickImage(source: ImageSource.gallery, imageQuality: 85)
            .then((x) {
          if (x != null) _upload(x.path, x.name);
        });
        return;
      case 'video':
        setState(() => _panelOpen = false);
        ImagePicker().pickVideo(source: ImageSource.gallery).then((x) {
          if (x != null) _upload(x.path, x.name);
        });
        return;
      case 'voice':
        setState(() => _panelOpen = false);
        _toggleRecord();
        return;
      case 'mention':
        _input.text += '@';
        setState(() => _panelOpen = false);
        return;
      case 'emoji':
        setState(() {
          _emojiOpen = !_emojiOpen;
          _panelOpen = false;
        });
        return;
      case 'favorite':
        setState(() => _panelOpen = false);
        _snack('长按消息即可收藏');
        return;
      default:
        setState(() => _panelOpen = false);
        _snack('暂不支持该功能');
    }
  }

  void _showCallMenu() {
    final call = context.read<CallState>();
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const ListTile(title: Text('发起通话', style: TextStyle(fontWeight: FontWeight.w600))),
            ListTile(
              leading: const Icon(Icons.call_outlined),
              title: const Text('语音通话'),
              onTap: () {
                Navigator.pop(ctx);
                if (call.busy) {
                  _snack('当前正在通话中');
                } else {
                  call.startCall(widget.conv, 'audio');
                }
              },
            ),
            ListTile(
              leading: const Icon(Icons.videocam_outlined),
              title: const Text('视频通话'),
              onTap: () {
                Navigator.pop(ctx);
                if (call.busy) {
                  _snack('当前正在通话中');
                } else {
                  call.startCall(widget.conv, 'video');
                }
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showConvMenu() {
    final app = context.read<AppState>();
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: Icon(widget.conv.pinned ? Icons.push_pin : Icons.push_pin_outlined),
              title: Text(widget.conv.pinned ? '取消置顶' : '置顶会话'),
              onTap: () {
                Navigator.pop(ctx);
                app.togglePinned(widget.conv);
              },
            ),
            ListTile(
              leading: Icon(widget.conv.muted
                  ? Icons.notifications_active_outlined
                  : Icons.notifications_off_outlined),
              title: Text(widget.conv.muted ? '开启提醒' : '消息免打扰'),
              onTap: () {
                Navigator.pop(ctx);
                app.toggleMuted(widget.conv);
              },
            ),
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
    final list = app.messages.where((m) => m.convId == widget.conv.convId).toList();
    if (list.length != _lastCount) {
      _lastCount = list.length;
      _scrollToBottom();
    }

    return Scaffold(
      backgroundColor: const Color(0xFFF0F1F5),
      body: Column(
        children: [
          _header(app),
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
          _composer(),
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
    final bubbleColor = mine ? const Color(0xFF4F6EF7) : const Color(0xFFE9E9EE);
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
          topLeft: const Radius.circular(14),
          topRight: const Radius.circular(14),
          bottomLeft: Radius.circular(mine ? 14 : 4),
          bottomRight: Radius.circular(mine ? 4 : 14),
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

  Widget _header(AppState app) {
    return Container(
      color: Colors.white,
      child: SafeArea(
        bottom: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(4, 6, 4, 6),
          child: Row(
            children: [
              IconButton(
                onPressed: () => Navigator.of(context).maybePop(),
                icon: const Icon(Icons.arrow_back_ios_new, size: 19, color: Color(0xFF1B2333)),
              ),
              Expanded(
                child: GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onTap: _showConvMenu,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(widget.conv.title,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                              fontSize: 17, fontWeight: FontWeight.w700, color: Color(0xFF1B2333))),
                      const SizedBox(height: 3),
                      Row(
                        children: [
                          Container(
                            width: 6,
                            height: 6,
                            decoration: BoxDecoration(
                              shape: BoxShape.circle,
                              color: app.connected ? const Color(0xFF25B389) : const Color(0xFFC3C9D6),
                            ),
                          ),
                          const SizedBox(width: 5),
                          Text(
                            widget.conv.isGroup ? '群聊' : (app.connected ? '在线 · WiFi' : '连接中…'),
                            style: TextStyle(
                                fontSize: 11.5,
                                color: app.connected ? const Color(0xFF25B389) : const Color(0xFF9AA5B8)),
                          ),
                          const Text(' ›', style: TextStyle(fontSize: 13, color: Color(0xFFAAB1BF))),
                        ],
                      ),
                    ],
                  ),
                ),
              ),
              if (!widget.conv.isGroup)
                IconButton(
                  onPressed: _showCallMenu,
                  icon: const Icon(Icons.call_outlined, color: Color(0xFF1B2333)),
                ),
              IconButton(
                onPressed: _showConvMenu,
                icon: const Icon(Icons.menu, color: Color(0xFF1B2333)),
              ),
            ],
          ),
        ),
      ),
    );
  }

  static const List<String> _emojis = [
    '😀', '😄', '😂', '😊', '😍', '🤔', '😎', '👍', '👏', '🙏', '🎉',
    '❤️', '🔥', '✅', '💡', '🚀', '📌', '😅', '🤝', '😴', '🙌'
  ];

  static const List<List<_PanelItem>> _panelPages = [
    [
      _PanelItem(Icons.call_outlined, '语音通话', 'call-audio'),
      _PanelItem(Icons.videocam_outlined, '视频通话', 'call-video'),
      _PanelItem(Icons.location_on_outlined, '位置', 'location'),
      _PanelItem(Icons.folder_outlined, '文件', 'file'),
      _PanelItem(Icons.card_giftcard, '红包', 'redpacket'),
      _PanelItem(Icons.account_balance_wallet_outlined, '转账', 'transfer'),
      _PanelItem(Icons.star_border, '收藏', 'favorite'),
      _PanelItem(Icons.screen_share_outlined, '屏幕分享', 'screenshare'),
    ],
    [
      _PanelItem(Icons.photo_library_outlined, '相册', 'image'),
      _PanelItem(Icons.video_library_outlined, '视频', 'video'),
      _PanelItem(Icons.mic_none, '语音消息', 'voice'),
      _PanelItem(Icons.alternate_email, '提及', 'mention'),
      _PanelItem(Icons.sentiment_satisfied_alt, '表情', 'emoji'),
    ],
  ];

  Widget _composer() {
    return Container(
      decoration: const BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.vertical(top: Radius.circular(18)),
      ),
      child: SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (_recording)
              Container(
                width: double.infinity,
                margin: const EdgeInsets.fromLTRB(12, 10, 12, 0),
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 9),
                decoration:
                    BoxDecoration(color: const Color(0xFFFFF1F1), borderRadius: BorderRadius.circular(12)),
                child: Row(
                  children: [
                    const Icon(Icons.graphic_eq, size: 17, color: Color(0xFFE5484D)),
                    const SizedBox(width: 8),
                    Text('正在录音 $_recSec″',
                        style: const TextStyle(fontSize: 13, color: Color(0xFFE5484D))),
                    const Spacer(),
                    GestureDetector(
                      onTap: _stopRecord,
                      child: const Text('发送',
                          style: TextStyle(
                              fontSize: 13,
                              fontWeight: FontWeight.w600,
                              color: Color(0xFFE5484D))),
                    ),
                  ],
                ),
              ),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 10, 12, 8),
              child: TextField(
                controller: _input,
                minLines: 1,
                maxLines: 1,
                textInputAction: TextInputAction.send,
                onSubmitted: (_) => _send(),
                onTap: () => setState(() {
                  _panelOpen = false;
                  _emojiOpen = false;
                }),
                style: const TextStyle(fontSize: 15),
                decoration: InputDecoration(
                  hintText: '发消息…',
                  hintStyle: const TextStyle(fontSize: 15, color: Color(0xFFADB3C1)),
                  isDense: true,
                  filled: true,
                  fillColor: Colors.white,
                  contentPadding: const EdgeInsets.symmetric(horizontal: 15, vertical: 11),
                  border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(22),
                      borderSide: const BorderSide(color: Color(0xFFE8EAF0))),
                  enabledBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(22),
                      borderSide: const BorderSide(color: Color(0xFFE8EAF0))),
                  focusedBorder: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(22),
                      borderSide: const BorderSide(color: Color(0xFFD7DBE4))),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(4, 0, 8, 6),
              child: Row(
                children: [
                  _tool(Icons.mic_none, _uploading ? null : _toggleRecord, active: _recording),
                  _tool(Icons.photo_library_outlined, _uploading ? null : () => _panelAction('image')),
                  _tool(Icons.photo_camera_outlined, _uploading ? null : _pickCamera),
                  _tool(Icons.auto_awesome, _toggleEmoji),
                  _tool(Icons.sentiment_satisfied_alt, _toggleEmoji),
                  const SizedBox(width: 6),
                  GestureDetector(
                    onTap: () => setState(() {
                      _panelOpen = !_panelOpen;
                      _emojiOpen = false;
                    }),
                    child: Container(
                      width: 38,
                      height: 38,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: const Color(0xFF2F7CF6),
                      ),
                      child: const Icon(Icons.close, size: 19, color: Colors.white),
                    ),
                  ),
                ],
              ),
            ),
            if (_emojiOpen) _emojiPanel(),
            if (_panelOpen) _gridPanel(),
          ],
        ),
      ),
    );
  }

  void _toggleEmoji() {
    setState(() {
      _emojiOpen = !_emojiOpen;
      _panelOpen = false;
    });
  }

  void _pickCamera() {
    ImagePicker().pickImage(source: ImageSource.camera, imageQuality: 85).then((x) {
      if (x != null) _upload(x.path, x.name);
    });
  }

  Widget _tool(IconData icon, VoidCallback? onTap, {bool active = false}) {
    return Expanded(
      child: IconButton(
        onPressed: onTap,
        iconSize: 26,
        padding: EdgeInsets.zero,
        constraints: const BoxConstraints(minHeight: 40),
        icon: Icon(icon, color: active ? const Color(0xFFE5484D) : const Color(0xFF20242C)),
      ),
    );
  }

  Widget _emojiPanel() {
    return Container(
      width: double.infinity,
      color: const Color(0xFFF1F3F8),
      padding: const EdgeInsets.fromLTRB(14, 10, 14, 8),
      child: Wrap(
        spacing: 12,
        runSpacing: 8,
        children: _emojis
            .map((e) => GestureDetector(
                  onTap: () => setState(() => _input.text += e),
                  child: Text(e, style: const TextStyle(fontSize: 26)),
                ))
            .toList(),
      ),
    );
  }

  Widget _gridPanel() {
    final items = _panelPages[_panelPage];
    return Container(
      width: double.infinity,
      color: const Color(0xFFF1F3F8),
      padding: const EdgeInsets.fromLTRB(10, 14, 10, 10),
      child: Column(
        children: [
          GridView.count(
            crossAxisCount: 4,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            mainAxisSpacing: 12,
            crossAxisSpacing: 8,
            childAspectRatio: 0.94,
            children: items
                .map((it) => InkWell(
                      onTap: () => _panelAction(it.key),
                      borderRadius: BorderRadius.circular(16),
                      child: Column(
                        children: [
                          Container(
                            height: 56,
                            width: double.infinity,
                            decoration: BoxDecoration(
                              color: Colors.white,
                              borderRadius: BorderRadius.circular(16),
                            ),
                            child: Icon(it.icon, size: 26, color: const Color(0xFF20242C)),
                          ),
                          const SizedBox(height: 7),
                          Text(it.label,
                              style: const TextStyle(fontSize: 12.5, color: Color(0xFF1F2329))),
                        ],
                      ),
                    ))
                .toList(),
          ),
          const SizedBox(height: 10),
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: List.generate(
              _panelPages.length,
              (i) => GestureDetector(
                onTap: () => setState(() => _panelPage = i),
                child: Container(
                  width: 6,
                  height: 6,
                  margin: const EdgeInsets.symmetric(horizontal: 3),
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    color: i == _panelPage ? const Color(0xFF3A3F4A) : const Color(0xFFC8CDD8),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _PanelItem {
  final IconData icon;
  final String label;
  final String key;
  const _PanelItem(this.icon, this.label, this.key);
}
