import 'dart:async';
import 'dart:io';

import 'package:audioplayers/audioplayers.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:image_picker/image_picker.dart';
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';
import 'package:record/record.dart';
import 'package:video_player/video_player.dart';

import '../app/app_state.dart';
import '../app/call_state.dart';
import '../core/api.dart';
import '../core/image_cache.dart';
import '../core/models.dart';
import '../core/util.dart';
import '../widgets/avatar.dart';
import 'group_info_page.dart';

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
  bool _isPlaying = false;
  Duration _pos = Duration.zero;
  Duration _total = Duration.zero;
  bool _userSeeking = false;
  bool _muted = false;
  double _speed = 1.0;
  StreamSubscription? _playerSub;
  StreamSubscription? _posSub;
  StreamSubscription? _durSub;
  // 工具栏面板
  bool _panelOpen = false;
  int _panelPage = 0;
  bool _emojiOpen = false;
  // 语音输入（按住说话）
  bool _voiceMode = false;
  bool _holdActive = false;
  bool _holdCancel = false;
  double _voiceStartY = 0;

  @override
  void initState() {
    super.initState();
    _app = context.read<AppState>();
    _app.setChatVisible(true);
    _playerSub = _player.onPlayerComplete.listen((_) {
      if (mounted) {
        setState(() {
          _isPlaying = false;
          _pos = Duration.zero;
        });
      }
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _app.openConv(widget.conv.convId);
    });
  }

  @override
  void dispose() {
    _playerSub?.cancel();
    _posSub?.cancel();
    _durSub?.cancel();
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
      // 授权/设备就绪前用户已松手：放弃本次录音
      if (_voiceMode && !_holdActive) {
        await _cancelRecord();
        return;
      }
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

  /// 取消录音：停止并删除临时文件，不发送
  Future<void> _cancelRecord() async {
    _recTimer?.cancel();
    _recTimer = null;
    String? path;
    try { path = await _recorder.stop(); } catch (_) {}
    if (mounted) setState(() { _recording = false; _recSec = 0; });
    if (path != null) {
      try { await File(path).delete(); } catch (_) {}
    }
  }

  void _toggleVoiceMode() {
    if (_voiceMode) {
      if (_recording) _cancelRecord();
      setState(() {
        _voiceMode = false;
        _holdActive = false;
        _holdCancel = false;
      });
    } else {
      setState(() {
        _voiceMode = true;
        _holdActive = false;
        _holdCancel = false;
        _emojiOpen = false;
        _panelOpen = false;
      });
    }
    FocusScope.of(context).unfocus();
  }

  void _onVoiceDown(PointerDownEvent e) {
    _holdActive = true;
    _holdCancel = false;
    _voiceStartY = e.position.dy;
    if (!_recording) _toggleRecord();
  }

  void _onVoiceMove(PointerMoveEvent e) {
    if (!_holdActive) return;
    final cancel = _voiceStartY - e.position.dy > 60;
    if (cancel != _holdCancel) setState(() => _holdCancel = cancel);
  }

  void _onVoiceUp() {
    if (!_holdActive) return;
    _holdActive = false;
    final cancel = _holdCancel;
    _holdCancel = false;
    if (!_recording) return;
    if (cancel) {
      _cancelRecord();
    } else {
      _stopRecord();
    }
  }

  Future<void> _togglePlay(Msg m) async {
    final extra = m.extra is Map ? Map<String, dynamic>.from(m.extra) : <String, dynamic>{};
    final fid = '${extra['fileId'] ?? ''}';
    if (fid.isEmpty) return;
    try {
      if (_playingId == m.msgId) {
        if (_isPlaying) {
          await _player.pause();
          if (mounted) setState(() => _isPlaying = false);
        } else {
          await _player.resume();
          if (mounted) setState(() => _isPlaying = true);
        }
        return;
      }
      await _player.stop();
      _posSub?.cancel();
      _durSub?.cancel();
      _posSub = _player.onPositionChanged.listen((p) {
        if (mounted && !_userSeeking) setState(() => _pos = p);
      });
      _durSub = _player.onDurationChanged.listen((d) {
        if (mounted && d.inMilliseconds > 0) setState(() => _total = d);
      });
      _pos = Duration.zero;
      _total = Duration(seconds: (extra['duration'] as num?)?.toInt() ?? 0);
      await _player.play(UrlSource(Api.fileUrl(fid)));
      await _player.setPlaybackRate(_speed);
      if (mounted) {
        setState(() {
          _playingId = m.msgId;
          _isPlaying = true;
        });
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('播放失败：$e')));
      }
    }
  }

  String _fmtDur(Duration d) {
    final mm = d.inMinutes;
    final ss = d.inSeconds % 60;
    return '$mm:${ss.toString().padLeft(2, '0')}';
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
    String localPath = '';
    try {
      // 复制一份到应用自有目录，发送后本地立即展示（系统缓存中的原文件随后清理）
      try {
        final dir = await _sentDir();
        final copy = File('${dir.path}/${DateTime.now().millisecondsSinceEpoch}_$name');
        await File(path).copy(copy.path);
        localPath = copy.path;
      } catch (_) {}
      final rec = await Api.upload('/files/upload', path, fields: {'convId': widget.conv.convId});
      final mime = '${rec['mimeType'] ?? ''}';
      final fname = '${rec['name'] ?? name}';
      // 上传接口可能只返回 application/octet-stream，按扩展名兜底判定媒体类型
      final ext = fname.contains('.') ? fname.toLowerCase().split('.').last : '';
      const imgExts = {'png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'heic', 'heif'};
      const videoExts = {'mp4', 'mov', 'm4v', 'webm', 'avi', 'mkv'};
      final type = mime.startsWith('image/') || imgExts.contains(ext)
          ? 'image'
          : (mime.startsWith('video/') || videoExts.contains(ext) ? 'video' : 'file');
      app.sendMedia(type, fname, {
        'fileId': rec['fileId'],
        'name': fname,
        'size': rec['size'] ?? 0,
        'mime': mime,
      }, localPath: localPath);
      // 清理 image_picker 在选择/拍摄时写入缓存的临时文件
      await _cleanPicked(path);
      await _pruneSent();
    } catch (e) {
      if (localPath.isNotEmpty) {
        try { await File(localPath).delete(); } catch (_) {}
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('上传失败：$e')));
      }
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<Directory> _sentDir() async {
    final dir = Directory('${(await getTemporaryDirectory()).path}/nexim_sent');
    if (!await dir.exists()) await dir.create(recursive: true);
    return dir;
  }

  /// 仅删除系统缓存目录下的临时文件（相册/相机选择产生），用户目录中的原文件不动
  Future<void> _cleanPicked(String path) async {
    try {
      final cacheRoot = (await getTemporaryDirectory()).path;
      if (!path.startsWith(cacheRoot) || path.contains('nexim_sent')) return;
      final f = File(path);
      if (await f.exists()) await f.delete();
    } catch (_) {}
  }

  /// 已发送图片的本地副本保留 7 天，避免缓存无限增长
  Future<void> _pruneSent() async {
    try {
      final dir = await _sentDir();
      final cutoff = DateTime.now().subtract(const Duration(days: 7));
      await for (final e in dir.list()) {
        if (e is! File) continue;
        try {
          if ((await e.stat()).modified.isBefore(cutoff)) await e.delete();
        } catch (_) {}
      }
    } catch (_) {}
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
        setState(() => _panelOpen = false);
        if (widget.conv.isGroup) {
          _showMentionPicker();
        } else {
          _input.text += '@';
        }
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

  Future<void> _openGroupInfo() async {
    final app = context.read<AppState>();
    final left = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => GroupInfoPage(groupId: widget.conv.peerId, groupName: widget.conv.title),
      ),
    );
    if (left == true && mounted) {
      await app.loadConvs();
      if (mounted) Navigator.of(context).maybePop();
    }
  }

  /// 群聊 @ 提及：先选群成员，再插入 @昵称
  Future<void> _showMentionPicker() async {
    List<Map<String, dynamic>> members = [];
    try {
      final data = await Api.get('/groups/${widget.conv.peerId}/members');
      if (data is List) {
        members = data.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
      }
    } catch (e) {
      if (mounted) _snack('加载成员失败：$e');
      return;
    }
    if (!mounted) return;
    final me = context.read<AppState>().myUid;
    final others = members.where((m) => '${m['uid'] ?? ''}' != me).toList();
    if (others.isEmpty) {
      _snack('暂无其他可提及的成员');
      return;
    }
    final picked = await showModalBottomSheet<Map<String, dynamic>>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Text('选择要提及的成员', style: TextStyle(fontWeight: FontWeight.w600)),
            ),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: others.length,
                itemBuilder: (_, i) {
                  final m = others[i];
                  return ListTile(
                    leading: Avatar(name: '${m['nickname'] ?? m['uid']}', url: '${m['avatar'] ?? ''}', size: 38),
                    title: Text('${m['nickname'] ?? m['uid']}', style: const TextStyle(fontSize: 15)),
                    onTap: () => Navigator.pop(ctx, m),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    );
    if (picked == null || !mounted) return;
    final nick = '${picked['nickname'] ?? picked['uid']}';
    final t = _input.text;
    _input.text = t.endsWith('@') ? '$t$nick ' : '$t@$nick ';
    _input.selection = TextSelection.collapsed(offset: _input.text.length);
  }

  void _showConvMenu() {
    final app = context.read<AppState>();
    showModalBottomSheet<void>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (widget.conv.isGroup)
              ListTile(
                leading: const Icon(Icons.groups_outlined),
                title: const Text('群聊信息'),
                onTap: () {
                  Navigator.pop(ctx);
                  _openGroupInfo();
                },
              ),
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

  // 历史消息可能因上传类型缺失被记为 file，按扩展名回退为图片渲染
  bool _isImageMsg(Msg m) {
    if (m.type == 'image') return true;
    if (m.type != 'file') return false;
    final ex = m.extra is Map ? Map<String, dynamic>.from(m.extra) : const <String, dynamic>{};
    final n = '${ex['name'] ?? m.content}'.toLowerCase();
    return RegExp(r'\.(png|jpe?g|gif|webp|bmp|heic|heif)$').hasMatch(n);
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
    if (m.type == 'call') {
      final ex = m.extra is Map ? Map<String, dynamic>.from(m.extra) : const <String, dynamic>{};
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Center(
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
            decoration: BoxDecoration(
              color: const Color(0xFFE9E9EE),
              borderRadius: BorderRadius.circular(14),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(ex['callType'] == 'video' ? Icons.videocam_outlined : Icons.call_outlined,
                    size: 14, color: Colors.grey),
                const SizedBox(width: 5),
                Text(m.content.isEmpty ? '通话已结束' : m.content,
                    style: const TextStyle(fontSize: 12, color: Colors.grey)),
              ],
            ),
          ),
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
    } else if (_isImageMsg(m)) {
      final fid = '${extra['fileId'] ?? ''}';
      final localPath = '${extra['localPath'] ?? ''}';
      const placeholder = Padding(
        padding: EdgeInsets.all(8),
        child: Text('[图片加载失败]', style: TextStyle(color: Colors.grey, fontSize: 12)),
      );
      Widget netImage() => CachedNetworkImage(
            imageUrl: Api.fileUrl(fid),
            cacheKey: 'file_$fid',
            cacheManager: AppImages.cache,
            width: 190,
            fit: BoxFit.cover,
            placeholder: (_, __) => const SizedBox(
              width: 190,
              height: 130,
              child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
            ),
            errorWidget: (_, __, ___) => placeholder,
          );
      content = fid.isEmpty
          ? Text('[图片]', style: TextStyle(color: textColor))
          : GestureDetector(
              onTap: () => _openImage(Api.fileUrl(fid), localPath, 'file_$fid'),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(8),
                // 自己刚发出的图片直接用本地副本显示，避免再下载一次
                child: (localPath.isNotEmpty && File(localPath).existsSync())
                    ? Image.file(
                        File(localPath),
                        width: 190,
                        fit: BoxFit.cover,
                        errorBuilder: (_, __, ___) => netImage(),
                      )
                    : netImage(),
              ),
            );
    } else if (m.type == 'video') {
      final fid = '${extra['fileId'] ?? ''}';
      content = InkWell(
        onTap: fid.isEmpty ? null : () => _openVideo(Api.fileUrl(fid), '${extra['name'] ?? '视频'}'),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.play_circle_fill, color: mine ? Colors.white : const Color(0xFF4F6EF7), size: 30),
            const SizedBox(width: 8),
            Text('视频 ${extra['name'] ?? ''}', style: TextStyle(color: textColor, fontSize: 13)),
          ],
        ),
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
      final current = _playingId == m.msgId;
      final playing = current && _isPlaying;
      final total = current && _total.inMilliseconds > 0 ? _total : Duration(seconds: dur);
      final pos = current ? _pos : Duration.zero;
      final maxMs = total.inMilliseconds > 0 ? total.inMilliseconds : 1000;
      final w = (MediaQuery.of(context).size.width * 0.58).clamp(200.0, 260.0).toDouble();
      const ink = Color(0xFF20242C);
      // 与 H5 一致的播放条：播放/暂停、0:00、可拖动进度、喇叭、更多
      content = Container(
        width: w,
        padding: const EdgeInsets.fromLTRB(6, 1, 2, 1),
        decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(9)),
        child: Row(
          children: [
            InkWell(
              onTap: fid.isEmpty ? null : () => _togglePlay(m),
              borderRadius: BorderRadius.circular(20),
              child: Padding(
                padding: const EdgeInsets.all(2),
                child: Icon(playing ? Icons.pause_rounded : Icons.play_arrow_rounded,
                    size: 22, color: ink),
              ),
            ),
            SizedBox(
              width: 36,
              child: Text(_fmtDur(pos),
                  textAlign: TextAlign.center,
                  style: const TextStyle(fontSize: 12.5, color: ink)),
            ),
            Expanded(
              child: SizedBox(
                height: 30,
                child: SliderTheme(
                  data: SliderTheme.of(context).copyWith(
                    trackHeight: 2,
                    thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 3.5),
                    overlayShape: const RoundSliderOverlayShape(overlayRadius: 9),
                    activeTrackColor: const Color(0xFF3C4048),
                    inactiveTrackColor: const Color(0xFFDADDE3),
                    thumbColor: const Color(0xFF3C4048),
                    disabledActiveTrackColor: const Color(0xFFDADDE3),
                    disabledInactiveTrackColor: const Color(0xFFDADDE3),
                    disabledThumbColor: const Color(0xFFDADDE3),
                  ),
                  child: Slider(
                    value: pos.inMilliseconds.clamp(0, maxMs).toDouble(),
                    max: maxMs.toDouble(),
                    onChangeStart: current ? (_) => _userSeeking = true : null,
                    onChanged: current
                        ? (v) => setState(() => _pos = Duration(milliseconds: v.round()))
                        : null,
                    onChangeEnd: current
                        ? (v) {
                            _userSeeking = false;
                            _player.seek(Duration(milliseconds: v.round()));
                          }
                        : null,
                  ),
                ),
              ),
            ),
            InkWell(
              borderRadius: BorderRadius.circular(20),
              onTap: () async {
                _muted = !_muted;
                await _player.setVolume(_muted ? 0 : 1);
                if (mounted) setState(() {});
              },
              child: Padding(
                padding: const EdgeInsets.all(4),
                child: Icon(_muted ? Icons.volume_off_rounded : Icons.volume_up_rounded,
                    size: 17, color: const Color(0xFF3C4048)),
              ),
            ),
            PopupMenuButton<double>(
              padding: EdgeInsets.zero,
              iconSize: 18,
              icon: const Icon(Icons.more_vert_rounded, size: 18, color: Color(0xFF3C4048)),
              itemBuilder: (_) => const [
                PopupMenuItem(value: 1.0, child: Text('1.0x')),
                PopupMenuItem(value: 1.5, child: Text('1.5x')),
                PopupMenuItem(value: 2.0, child: Text('2.0x')),
              ],
              onSelected: (s) async {
                setState(() => _speed = s);
                await _player.setPlaybackRate(s);
              },
            ),
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
          if (mine) ...[
            const SizedBox(width: 8),
            Avatar(name: app.myName, url: '${app.user?['avatar'] ?? ''}', size: 34),
          ],
        ],
      ),
    );
  }

  void _openImage(String url, [String localPath = '', String cacheKey = '']) {
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => _ImageViewerPage(url: url, localPath: localPath, cacheKey: cacheKey),
    ));
  }

  void _openVideo(String url, String title) {
    Navigator.of(context).push(MaterialPageRoute(builder: (_) => _VideoPlayerPage(url: url, title: title)));
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
              if (widget.conv.isGroup)
                IconButton(
                  onPressed: _openGroupInfo,
                  icon: const Icon(Icons.groups_outlined, color: Color(0xFF1B2333)),
                )
              else
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

  List<List<_PanelItem>> get _panelPages => [
        [
          // 群聊不支持 1 对 1 通话，直接隐藏入口
          if (!widget.conv.isGroup) ...[
            const _PanelItem(Icons.call_outlined, '语音通话', 'call-audio'),
            const _PanelItem(Icons.videocam_outlined, '视频通话', 'call-video'),
          ],
          const _PanelItem(Icons.folder_outlined, '文件', 'file'),
          const _PanelItem(Icons.star_border, '收藏', 'favorite'),
          const _PanelItem(Icons.photo_library_outlined, '相册', 'image'),
          const _PanelItem(Icons.video_library_outlined, '视频', 'video'),
        ],
        [
          const _PanelItem(Icons.mic_none, '语音消息', 'voice'),
          const _PanelItem(Icons.alternate_email, '提及', 'mention'),
          const _PanelItem(Icons.sentiment_satisfied_alt, '表情', 'emoji'),
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
            // 录音提示（与 H5 的 HUD 一致）
            if (_recording)
              Center(
                child: Container(
                  margin: const EdgeInsets.only(top: 10),
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 9),
                  decoration: BoxDecoration(
                    color: _holdCancel ? const Color(0xF2E5484D) : const Color(0xEB1F2937),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.graphic_eq, size: 16, color: Colors.white),
                      const SizedBox(width: 9),
                      Text('$_recSec″',
                          style: const TextStyle(
                              fontSize: 12, fontWeight: FontWeight.w700, color: Colors.white)),
                      const SizedBox(width: 9),
                      Text(_holdCancel ? '松开 取消' : '松开 发送 · 上滑 取消',
                          style: const TextStyle(fontSize: 12, color: Colors.white70)),
                    ],
                  ),
                ),
              ),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 10, 12, 8),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  Expanded(
                    child: _voiceMode
                        ? _holdTalkBar()
                        : TextField(
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
                  const SizedBox(width: 8),
                  // 发送按钮：随输入内容启用/禁用
                  ValueListenableBuilder<TextEditingValue>(
                    valueListenable: _input,
                    builder: (_, v, __) {
                      final on = v.text.trim().isNotEmpty;
                      return GestureDetector(
                        onTap: on ? _send : null,
                        child: Container(
                          width: 38,
                          height: 38,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            color: on ? const Color(0xFF2F7CF6) : const Color(0xFFC9D3E3),
                          ),
                          child: const Icon(Icons.send_rounded, size: 18, color: Colors.white),
                        ),
                      );
                    },
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(0, 4, 0, 10),
              child: Row(
                children: [
                  _tool(Icons.mic_none, _uploading ? null : _toggleVoiceMode,
                      active: _voiceMode || _recording),
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
                      width: 26,
                      height: 26,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: Colors.white,
                        border: Border.all(color: const Color(0xFF1A1A1A), width: 1.2),
                      ),
                      // 默认显示 +，展开面板后变为 ×；与左侧工具图标同尺寸
                      child: Icon(_panelOpen ? Icons.close : Icons.add,
                          size: 16, color: const Color(0xFF1A1A1A)),
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

  /// 输入区「按住说话」，样式与 H5 一致
  Widget _holdTalkBar() {
    final recording = _recording;
    final cancel = _holdCancel;
    final Color bg;
    final Color fg;
    if (recording && cancel) {
      bg = const Color(0xFFFFECEE);
      fg = const Color(0xFFE5484D);
    } else if (recording) {
      bg = const Color(0xFFE6F2FF);
      fg = const Color(0xFF2F7CF6);
    } else {
      bg = const Color(0xFFF5F6FA);
      fg = const Color(0xFF57607A);
    }
    return Listener(
      onPointerDown: _onVoiceDown,
      onPointerMove: _onVoiceMove,
      onPointerUp: (_) => _onVoiceUp(),
      onPointerCancel: (_) => _onVoiceUp(),
      child: Container(
        height: 42,
        alignment: Alignment.center,
        decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(22)),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.mic_none_rounded, size: 19, color: fg),
            const SizedBox(width: 6),
            Text(
              recording ? (cancel ? '松开 取消' : '松开 发送') : '按住 说话',
              style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: fg),
            ),
          ],
        ),
      ),
    );
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

/// 图片全屏查看：双击/双指缩放，点击返回
class _ImageViewerPage extends StatefulWidget {
  final String url;
  final String localPath;
  final String cacheKey;
  const _ImageViewerPage({required this.url, this.localPath = '', this.cacheKey = ''});

  @override
  State<_ImageViewerPage> createState() => _ImageViewerPageState();
}

class _ImageViewerPageState extends State<_ImageViewerPage> {
  Widget _net() {
    final key = widget.cacheKey.isEmpty ? widget.url : widget.cacheKey;
    return CachedNetworkImage(
      imageUrl: widget.url,
      cacheKey: key,
      cacheManager: AppImages.cache,
      fit: BoxFit.contain,
      placeholder: (_, __) => const Center(child: CircularProgressIndicator(color: Colors.white70)),
      errorWidget: (_, __, ___) => const Center(
        child: Text('图片加载失败', style: TextStyle(color: Colors.white70, fontSize: 14)),
      ),
    );
  }

  Widget _image() {
    if (widget.localPath.isNotEmpty && File(widget.localPath).existsSync()) {
      return Image.file(
        File(widget.localPath),
        fit: BoxFit.contain,
        errorBuilder: (_, __, ___) => _net(),
      );
    }
    return _net();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        iconTheme: const IconThemeData(color: Colors.white),
      ),
      extendBodyBehindAppBar: true,
      body: InteractiveViewer(
        minScale: 1,
        maxScale: 4,
        child: _image(),
      ),
    );
  }
}

/// 视频播放：点击画面播放/暂停，底部可拖动进度
class _VideoPlayerPage extends StatefulWidget {
  final String url;
  final String title;
  const _VideoPlayerPage({required this.url, required this.title});

  @override
  State<_VideoPlayerPage> createState() => _VideoPlayerPageState();
}

class _VideoPlayerPageState extends State<_VideoPlayerPage> {
  VideoPlayerController? _controller;
  bool _failed = false;

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final controller = VideoPlayerController.networkUrl(Uri.parse(widget.url));
    _controller = controller;
    controller.addListener(_onTick);
    try {
      await controller.initialize();
      await controller.setLooping(false);
      await controller.play();
      if (mounted) setState(() {});
    } catch (_) {
      if (mounted) setState(() => _failed = true);
    }
  }

  void _onTick() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    _controller?.removeListener(_onTick);
    _controller?.dispose();
    super.dispose();
  }

  String _fmt(Duration d) {
    final m = d.inMinutes.toString().padLeft(2, '0');
    final s = (d.inSeconds % 60).toString().padLeft(2, '0');
    return '$m:$s';
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    final ready = controller != null && controller.value.isInitialized;
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        iconTheme: const IconThemeData(color: Colors.white),
        title: Text(widget.title,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(color: Colors.white, fontSize: 15)),
      ),
      extendBodyBehindAppBar: true,
      body: _failed
          ? const Center(child: Text('视频加载失败', style: TextStyle(color: Colors.white70, fontSize: 14)))
          : Center(
              child: !ready
                  ? const CircularProgressIndicator(color: Colors.white70)
                  : GestureDetector(
                      onTap: () {
                        if (controller.value.isPlaying) {
                          controller.pause();
                        } else {
                          controller.play();
                        }
                        setState(() {});
                      },
                      child: Stack(
                        alignment: Alignment.center,
                        children: [
                          AspectRatio(
                            aspectRatio: controller.value.aspectRatio <= 0 ? 16 / 9 : controller.value.aspectRatio,
                            child: VideoPlayer(controller),
                          ),
                          if (!controller.value.isPlaying)
                            const Icon(Icons.play_circle_fill, color: Colors.white70, size: 64),
                        ],
                      ),
                    ),
            ),
      bottomNavigationBar: ready
          ? SafeArea(
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                child: Row(
                  children: [
                    Text(_fmt(controller.value.position),
                        style: const TextStyle(color: Colors.white70, fontSize: 12)),
                    Expanded(
                      child: VideoProgressIndicator(controller, allowScrubbing: true, padding: const EdgeInsets.symmetric(horizontal: 10)),
                    ),
                    Text(_fmt(controller.value.duration),
                        style: const TextStyle(color: Colors.white70, fontSize: 12)),
                  ],
                ),
              ),
            )
          : null,
    );
  }
}
